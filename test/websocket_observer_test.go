package rrpc_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/netgarden/rrpc"
	"github.com/netgarden/rrpc/test/modules/echo"
	echoImpl "github.com/netgarden/rrpc/test/modules/echo/impl"
)

// A ResponseObserver wraps the ResponseWriter to record status and size; a
// WebSocket upgrade must still be able to hijack the connection through that
// wrapper, and the observer must see the upgrade (101) once the stream ends.
func TestObserverDoesNotBreakWebSocketUpgrade(t *testing.T) {
	echoModule := echo.NewModule()
	echoModule.SetEchoService(echoImpl.NewEchoService())

	var mu sync.Mutex
	var seen []rrpc.ResponseInfo
	done := make(chan struct{}, 1)

	rrpcServer := rrpc.NewServer()
	rrpcServer.Observe(func(r *http.Request, info rrpc.ResponseInfo) {
		mu.Lock()
		seen = append(seen, info)
		mu.Unlock()
		done <- struct{}{}
	})
	if err := rrpcServer.RegisterModule(echoModule); err != nil {
		t.Fatalf("RegisterModule: %v", err)
	}
	srv := httptest.NewServer(rrpcServer)
	t.Cleanup(srv.Close)
	client := echo.NewClient(srv.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.Echo.Chat(ctx)
	if err != nil {
		t.Fatalf("Chat through an observed server: %v", err)
	}
	if err := stream.Send(echo.ChatMessage{Text: "hello"}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	reply, err := stream.Recv()
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}
	if reply.Text != "hello" {
		t.Errorf("Recv = %+v", reply)
	}
	stream.Close(1000, "")

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("the observer was never called for the finished WebSocket")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 1 || seen[0].Status != http.StatusSwitchingProtocols {
		t.Errorf("observed %+v, want one request with status 101", seen)
	}
	if seen[0].Route == "" {
		t.Error("the WebSocket route should be reported by its template")
	}
}
