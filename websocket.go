package rrpc

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// UpgradeOptions configures a WebSocket upgrade. Leaving Subprotocols nil
// does NOT mean "no subprotocol negotiation" - if the client's handshake
// offered any (e.g. maf/rrpc-auth's "bearer, <token>" trick for carrying
// an auth token, since a browser's native WebSocket constructor can't set
// arbitrary headers on the handshake request), upgrade mirrors them back
// automatically. This isn't optional politeness: per RFC 6455 §4.1, a
// client that offered subprotocols MUST fail the connection if the
// server's response doesn't select one of them - every real
// implementation (browsers, the "ws" npm package, Go's x/net/websocket)
// enforces this, even though nothing in a raw HTTP Upgrade response makes
// the omission look wrong (curl, for instance, won't complain). Set
// Subprotocols explicitly only when a handler needs to require or choose
// among specific values itself.
type UpgradeOptions struct {
	Subprotocols []string
}

// duplex is the shared implementation behind Stream/InStream/OutStream -
// one *websocket.Conn plus the connection-scoped context (cancelled on
// disconnect or server shutdown). Stream/InStream/OutStream each embed it
// unexported and expose only the method subset appropriate to their
// direction, so the generated server interface itself enforces what a
// handler can do (an @OutputStream handler's parameter has no Recv to
// accidentally call, and vice versa).
type duplex struct {
	conn *websocket.Conn
	ctx  context.Context
}

// Context returns the connection-scoped context - use this, not the
// original request context, in a long-running Recv/Send loop.
func (d *duplex) Context() context.Context { return d.ctx }

// Close closes the underlying connection. Safe to call multiple times.
func (d *duplex) Close(code websocket.StatusCode, reason string) error {
	return d.conn.Close(code, reason)
}

// CloseGracefully closes the connection with a normal-closure status - the
// generated handler wrapper for every streaming method defers this once
// Upgrade succeeds, specifically so a handler that returns normally (all
// data sent, done) ends the connection with a proper WS close frame
// instead of leaving that to Context()'s fallback. That fallback (see
// Context's doc comment and upgrade's use of the request context) only
// exists to eventually reclaim a connection whose handler forgot to close
// it - it does so via CloseRead's background reader observing context
// cancellation, which lands on Conn's internal abrupt teardown, not a
// graceful close, and shows up client-side as a "closed abnormally"
// error even though every byte was already delivered. Safe to call
// multiple times, same as Close.
func (d *duplex) CloseGracefully() error {
	return d.conn.Close(websocket.StatusNormalClosure, "")
}

// closeReadOnly is applied to a send-only duplex (OutStream/RawOutStream
// server-side, ClientSendStream/ClientRawSendStream client-side) - none
// of those ever call Recv/Read, but coder/websocket requires *someone* to
// keep reading so ping/pong/close control frames get processed (see
// Conn.CloseRead's doc comment: "you must always read from the
// connection... otherwise control frames will not be handled"). Without
// this, Close never receives the peer's close acknowledgment (it just
// times out) and Context() never cancels on peer disconnect - both of
// which OutStream/ClientSendStream's own doc comments promise work.
func (d duplex) closeReadOnly() duplex {
	d.ctx = d.conn.CloseRead(d.ctx)
	return d
}

// outgoingEnvelope is the wire shape of every frame Send/SendError writes
// on a typed, Send-capable stream (Stream/OutStream) - exactly one of
// Data or Error is ever set, letting the client distinguish a normal
// message from a typed rrpc error without needing a separate framing
// scheme. Incoming (client->server) frames are NOT enveloped - there's no
// "client sends a typed error" concept, so Recv reads bare JSON.
type outgoingEnvelope[Out any] struct {
	Data  *Out       `json:"data,omitempty"`
	Error *RRPCError `json:"error,omitempty"`
}

// Stream is a bidirectional typed message stream - "@Stream".
type Stream[In, Out any] struct{ duplex }

// Recv reads and JSON-decodes the next client->server frame. A peer close
// surfaces as an error - use websocket.CloseStatus(err) to distinguish a
// normal close from an abnormal one, same as using coder/websocket
// directly.
func (s *Stream[In, Out]) Recv() (In, error) {
	var msg In
	err := wsjson.Read(s.ctx, s.conn, &msg)
	return msg, err
}

// Send writes msg as the next server->client frame.
func (s *Stream[In, Out]) Send(msg Out) error {
	return wsjson.Write(s.ctx, s.conn, outgoingEnvelope[Out]{Data: &msg})
}

// SendError writes err (normalized to an RRPCError - see toRRPCError) as
// the next server->client frame, distinguishable from a normal Send'd
// message by the client.
func (s *Stream[In, Out]) SendError(err error) error {
	rpcErr := toRRPCError(err)
	return wsjson.Write(s.ctx, s.conn, outgoingEnvelope[Out]{Error: &rpcErr})
}

// InStream is a client->server-only typed message stream -
// "@InputStream". The server only receives; there's no Send/SendError -
// no existing "client sends a typed error to the server" concept to
// extend.
type InStream[In any] struct{ duplex }

func (s *InStream[In]) Recv() (In, error) {
	var msg In
	err := wsjson.Read(s.ctx, s.conn, &msg)
	return msg, err
}

// OutStream is a server->client-only typed message stream -
// "@OutputStream". The server only sends; it has no typed Recv, but can
// still observe disconnect via Context() cancellation.
type OutStream[Out any] struct{ duplex }

func (s *OutStream[Out]) Send(msg Out) error {
	return wsjson.Write(s.ctx, s.conn, outgoingEnvelope[Out]{Data: &msg})
}

func (s *OutStream[Out]) SendError(err error) error {
	rpcErr := toRRPCError(err)
	return wsjson.Write(s.ctx, s.conn, outgoingEnvelope[Out]{Error: &rpcErr})
}

// RawStream, RawInStream, RawOutStream are the raw-byte-stream
// counterparts of Stream/InStream/OutStream, generated instead of the
// typed generics above when a streaming method's direction is declared
// with "bytes" rather than a struct type. Each embeds the corresponding
// stdlib interface directly via websocket.NetConn (which already handles
// buffering/chunking across WebSocket frame boundaries), so it can be
// passed straight into io.Copy, wrapped in a bufio.Scanner, or handed to
// any other io.Reader/io.Writer-consuming code.
type RawStream struct {
	io.ReadWriter
	duplex
}

type RawInStream struct {
	io.Reader
	duplex
}

type RawOutStream struct {
	io.Writer
	duplex
}

// UpgradeStream/UpgradeInStream/UpgradeOutStream and their Raw
// counterparts take over the HTTP connection behind ctx via
// coder/websocket. Each may be called at most once per Context.
// ctx's normal HTTP response path is no longer usable after calling one
// of these - even on failure, coder/websocket's Accept has already
// written its own response to ctx.Response() (see its doc comment), so
// the framework will not attempt to write a JSON error response for
// whatever error the calling MethodHandlerFunc eventually returns (see
// Context.Hijacked, coreHandle/recoverPanic's hijacked-aware
// error-write skip) - callers should not try to write to ctx.Response()
// after calling one of these, successful or not.

func UpgradeStream[In, Out any](ctx *Context, opts UpgradeOptions) (*Stream[In, Out], error) {
	d, err := upgrade(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &Stream[In, Out]{duplex: d}, nil
}

func UpgradeInStream[In any](ctx *Context, opts UpgradeOptions) (*InStream[In], error) {
	d, err := upgrade(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &InStream[In]{duplex: d}, nil
}

func UpgradeOutStream[Out any](ctx *Context, opts UpgradeOptions) (*OutStream[Out], error) {
	d, err := upgrade(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &OutStream[Out]{duplex: d.closeReadOnly()}, nil
}

func UpgradeRawStream(ctx *Context, opts UpgradeOptions) (*RawStream, error) {
	d, err := upgrade(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &RawStream{ReadWriter: websocket.NetConn(d.ctx, d.conn, websocket.MessageBinary), duplex: d}, nil
}

func UpgradeRawInStream(ctx *Context, opts UpgradeOptions) (*RawInStream, error) {
	d, err := upgrade(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &RawInStream{Reader: websocket.NetConn(d.ctx, d.conn, websocket.MessageBinary), duplex: d}, nil
}

func UpgradeRawOutStream(ctx *Context, opts UpgradeOptions) (*RawOutStream, error) {
	d, err := upgrade(ctx, opts)
	if err != nil {
		return nil, err
	}
	d = d.closeReadOnly()
	return &RawOutStream{Writer: websocket.NetConn(d.ctx, d.conn, websocket.MessageBinary), duplex: d}, nil
}

func upgrade(ctx *Context, opts UpgradeOptions) (duplex, error) {
	subprotocols := opts.Subprotocols
	if len(subprotocols) == 0 {
		subprotocols = offeredSubprotocols(ctx.Request().Header)
	}
	conn, err := websocket.Accept(ctx.Response(), ctx.Request(), &websocket.AcceptOptions{
		Subprotocols: subprotocols,
	})
	// websocket.Accept writes its own response to ctx.Response() on every
	// outcome, success or failure (see its doc comment: "Accept will
	// write a response to w on all errors") - mark hijacked regardless of
	// err, so the framework never attempts to also write a JSON error on
	// top of whatever Accept already sent (which would otherwise surface
	// as a "superfluous response.WriteHeader call" and a corrupted
	// response body).
	ctx.markHijacked()
	if err != nil {
		return duplex{}, err
	}
	return duplex{conn: conn, ctx: ctx.Request().Context()}, nil
}

// offeredSubprotocols parses every Sec-WebSocket-Protocol header value
// into individual tokens, mirroring how coder/websocket's own (unexported)
// headerTokens parses them when selecting a subprotocol - so what upgrade
// offers back to Accept lines up exactly with what it will look for.
func offeredSubprotocols(h http.Header) []string {
	var tokens []string
	for _, v := range h.Values("Sec-WebSocket-Protocol") {
		for _, t := range strings.Split(v, ",") {
			if t = strings.TrimSpace(t); t != "" {
				tokens = append(tokens, t)
			}
		}
	}
	return tokens
}
