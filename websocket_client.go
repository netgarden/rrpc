package rrpc

import (
	"context"
	"io"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// DialOptions configures a client-side streaming connection - the dialing
// counterpart of UpgradeOptions. Bare today; grows a token/subprotocol
// option later if/when the generated non-streaming Go client adopts one,
// rather than inventing one speculatively here.
type DialOptions struct {
	Subprotocols []string
}

// ClientStream, ClientSendStream, ClientRecvStream are the generated Go
// client's counterparts of Stream/InStream/OutStream - used by one Go
// program calling another rrpc-based Go service directly (via the
// generated *ServiceClientHandler methods), independent of the browser/TS
// client. Type parameter naming stays consistent with the .rrpc
// declaration regardless of which side's code you're reading (In is
// always the client->server type, Out always server->client) - only the
// available method set flips between server (accepting an inbound
// connection) and client (dialing an outbound one).
type ClientStream[In, Out any] struct{ duplex }

func (s *ClientStream[In, Out]) Send(msg In) error {
	return wsjson.Write(s.ctx, s.conn, msg)
}

// Recv reads the next server->client frame. A received {"error": ...}
// envelope surfaces here as a returned Go error (built from the same
// RRPCError shape the server sends) - the natural Go equivalent of the TS
// client's message-iterator throwing on an error frame.
func (s *ClientStream[In, Out]) Recv() (Out, error) {
	var env outgoingEnvelope[Out]
	if err := wsjson.Read(s.ctx, s.conn, &env); err != nil {
		var zero Out
		return zero, normalClosureAsEOF(err)
	}
	if env.Error != nil {
		var zero Out
		return zero, *env.Error
	}
	if env.Data == nil {
		var zero Out
		return zero, nil
	}
	return *env.Data, nil
}

// ClientSendStream is the client side of "@InputStream" - the client is
// the one sending, so it only has Send.
type ClientSendStream[In any] struct{ duplex }

func (s *ClientSendStream[In]) Send(msg In) error {
	return wsjson.Write(s.ctx, s.conn, msg)
}

// ClientRecvStream is the client side of "@OutputStream" - the client is
// the one receiving, so it only has Recv (with the same error-frame
// surfacing as ClientStream.Recv).
type ClientRecvStream[Out any] struct{ duplex }

func (s *ClientRecvStream[Out]) Recv() (Out, error) {
	var env outgoingEnvelope[Out]
	if err := wsjson.Read(s.ctx, s.conn, &env); err != nil {
		var zero Out
		return zero, normalClosureAsEOF(err)
	}
	if env.Error != nil {
		var zero Out
		return zero, *env.Error
	}
	if env.Data == nil {
		var zero Out
		return zero, nil
	}
	return *env.Data, nil
}

// normalClosureAsEOF maps a graceful WebSocket close (the server side's
// CloseGracefully, called once a streaming handler returns - see
// websocket.go) to io.EOF, the idiomatic Go signal for "stream ended, no
// more data" that callers naturally check for via errors.Is(err, io.EOF).
// Without this, a deliberate graceful close surfaces as a raw
// *websocket.CloseError instead, indistinguishable at a glance from any
// other stream failure.
func normalClosureAsEOF(err error) error {
	if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
		return io.EOF
	}
	return err
}

// ClientRawStream, ClientRawSendStream, ClientRawRecvStream are the
// dialing counterparts of RawStream/RawInStream/RawOutStream, used for
// streaming methods whose direction is declared "bytes" rather than a
// struct type.
type ClientRawStream struct {
	io.ReadWriter
	duplex
}

type ClientRawSendStream struct {
	io.Writer
	duplex
}

type ClientRawRecvStream struct {
	io.Reader
	duplex
}

// DialStream/DialSendStream/DialRecvStream and their Raw counterparts
// dial url as a WebSocket client via coder/websocket. Each returns once
// the handshake completes.

func DialStream[In, Out any](ctx context.Context, url string, opts DialOptions) (*ClientStream[In, Out], error) {
	d, err := dial(ctx, url, opts)
	if err != nil {
		return nil, err
	}
	return &ClientStream[In, Out]{duplex: d}, nil
}

func DialSendStream[In any](ctx context.Context, url string, opts DialOptions) (*ClientSendStream[In], error) {
	d, err := dial(ctx, url, opts)
	if err != nil {
		return nil, err
	}
	return &ClientSendStream[In]{duplex: d.closeReadOnly()}, nil
}

func DialRecvStream[Out any](ctx context.Context, url string, opts DialOptions) (*ClientRecvStream[Out], error) {
	d, err := dial(ctx, url, opts)
	if err != nil {
		return nil, err
	}
	return &ClientRecvStream[Out]{duplex: d}, nil
}

func DialRawStream(ctx context.Context, url string, opts DialOptions) (*ClientRawStream, error) {
	d, err := dial(ctx, url, opts)
	if err != nil {
		return nil, err
	}
	return &ClientRawStream{ReadWriter: websocket.NetConn(d.ctx, d.conn, websocket.MessageBinary), duplex: d}, nil
}

func DialRawSendStream(ctx context.Context, url string, opts DialOptions) (*ClientRawSendStream, error) {
	d, err := dial(ctx, url, opts)
	if err != nil {
		return nil, err
	}
	d = d.closeReadOnly()
	return &ClientRawSendStream{Writer: websocket.NetConn(d.ctx, d.conn, websocket.MessageBinary), duplex: d}, nil
}

func DialRawRecvStream(ctx context.Context, url string, opts DialOptions) (*ClientRawRecvStream, error) {
	d, err := dial(ctx, url, opts)
	if err != nil {
		return nil, err
	}
	return &ClientRawRecvStream{Reader: websocket.NetConn(d.ctx, d.conn, websocket.MessageBinary), duplex: d}, nil
}

func dial(ctx context.Context, url string, opts DialOptions) (duplex, error) {
	conn, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{
		Subprotocols: opts.Subprotocols,
	})
	if err != nil {
		return duplex{}, err
	}
	// Unlike the server side (whose connection-scoped context comes from
	// the original *http.Request and is cancelled on disconnect
	// automatically), the client dial's ctx is caller-supplied and may
	// have nothing to do with the connection's actual lifetime - use it
	// as-is; callers wanting cancellation-on-disconnect should derive
	// their own context and cancel it themselves once Send/Recv starts
	// erroring.
	return duplex{conn: conn, ctx: ctx}, nil
}
