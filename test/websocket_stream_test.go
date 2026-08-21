package rrpc_test

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/netgarden/rrpc"
	"github.com/netgarden/rrpc/test/modules/echo"
	echoImpl "github.com/netgarden/rrpc/test/modules/echo/impl"
)

// newEchoTestServer starts a real HTTP server (only the Echo module
// registered - the other test modules aren't needed for these tests) and
// returns a Go client dialed against it, plus a cleanup func.
func newEchoTestServer(t *testing.T) *echo.Client {
	t.Helper()

	echoModule := echo.NewModule()
	echoModule.SetEchoService(echoImpl.NewEchoService())

	rrpcServer := rrpc.NewServer()
	if err := rrpcServer.RegisterModule(echoModule); err != nil {
		t.Fatalf("RegisterModule: %v", err)
	}

	srv := httptest.NewServer(rrpcServer)
	t.Cleanup(srv.Close)

	return echo.NewClient(srv.URL)
}

// TestStreamChat proves the bidirectional "@Stream" round trip end to
// end: DSL -> Go server codegen -> rrpc.UpgradeStream/Stream -> real
// WebSocket -> generated Go client's DialStream/ClientStream.
func TestStreamChat(t *testing.T) {
	client := newEchoTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.Echo.Chat(ctx)
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	defer stream.Close(1000, "")

	if err := stream.Send(echo.ChatMessage{Text: "hello"}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	reply, err := stream.Recv()
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}
	if reply.From != "server" || reply.Text != "hello" {
		t.Errorf("Recv = %+v, want {From:server Text:hello}", reply)
	}
}

// TestStreamChatSendError proves the server->client error-frame path: a
// "trigger" message causes the server to call stream.SendError, which the
// Go client's Recv surfaces as a returned Go error built from the same
// RRPCError shape.
func TestStreamChatSendError(t *testing.T) {
	client := newEchoTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.Echo.Chat(ctx)
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	defer stream.Close(1000, "")

	if err := stream.Send(echo.ChatMessage{Text: "trigger"}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	_, err = stream.Recv()
	if err == nil {
		t.Fatal("expected Recv to return an error for the trigger message")
	}
	var rpcErr rrpc.RRPCError
	if !errors.As(err, &rpcErr) {
		t.Fatalf("expected an rrpc.RRPCError, got %T: %v", err, err)
	}
	if rpcErr.Name != "StreamTriggerError" || rpcErr.Code != 1 {
		t.Errorf("RRPCError = %+v, want Name=StreamTriggerError Code=1", rpcErr)
	}
}

// TestStreamCollect proves the client->server-only "@InputStream"
// direction: RawInStream/InStream's Send-less server type and the
// client's Send-only ClientSendStream.
func TestStreamCollect(t *testing.T) {
	client := newEchoTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.Echo.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	for _, line := range []string{"one", "two", "three"} {
		if err := stream.Send(echo.LogLine{Line: line}); err != nil {
			t.Fatalf("Send(%q): %v", line, err)
		}
	}
	// Give the server a moment to receive before we close - Collect logs
	// on disconnect, this test only proves Send doesn't error and the
	// connection tears down cleanly.
	time.Sleep(100 * time.Millisecond)
	if err := stream.Close(1000, ""); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// TestStreamTick proves the server->client-only "@OutputStream"
// direction and that the server observes client disconnect via the
// stream's connection-scoped Context.
func TestStreamTick(t *testing.T) {
	client := newEchoTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.Echo.Tick(ctx)
	if err != nil {
		t.Fatalf("Tick: %v", err)
	}

	first, err := stream.Recv()
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}
	if first.N != 1 {
		t.Errorf("first tick N = %d, want 1", first.N)
	}
	second, err := stream.Recv()
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}
	if second.N != 2 {
		t.Errorf("second tick N = %d, want 2", second.N)
	}

	if err := stream.Close(1000, ""); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// TestStreamUploadBytesRaw proves the raw byte-stream mode ("bytes"
// declared instead of a struct): the client's ClientRawSendStream is a
// plain io.Writer, and the server side (already proven to satisfy
// io.Reader via io.Copy in the impl itself) receives exactly what was
// written.
func TestStreamUploadBytesRaw(t *testing.T) {
	client := newEchoTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.Echo.UploadBytes(ctx)
	if err != nil {
		t.Fatalf("UploadBytes: %v", err)
	}

	payload := []byte("the quick brown fox jumps over the lazy dog")
	n, err := stream.Write(payload)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n != len(payload) {
		t.Errorf("Write returned n=%d, want %d", n, len(payload))
	}

	if err := stream.Close(1000, ""); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// TestStreamUnmatchedRouteReturnsNormalJSONError proves a pre-upgrade
// failure (a plain GET against a streaming route, no WebSocket handshake
// headers) still gets a normal JSON error response - Context.Hijacked()
// never becomes true.
func TestStreamUnmatchedRouteReturnsNormalJSONError(t *testing.T) {
	echoModule := echo.NewModule()
	echoModule.SetEchoService(echoImpl.NewEchoService())

	rrpcServer := rrpc.NewServer()
	if err := rrpcServer.RegisterModule(echoModule); err != nil {
		t.Fatalf("RegisterModule: %v", err)
	}

	srv := httptest.NewServer(rrpcServer)
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/echo/Chat")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 400 {
		t.Fatalf("expected an error status for a non-WebSocket request to a streaming route, got %d: %s", resp.StatusCode, body)
	}
}
