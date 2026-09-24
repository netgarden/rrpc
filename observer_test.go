package rrpc_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/netgarden/rrpc"
	"github.com/netgarden/rrpc/internal/testmodule"
)

type observed struct {
	method string
	info   rrpc.ResponseInfo
}

// newObservedServer is newTestServer plus an observer collecting every request.
func newObservedServer(t *testing.T) (*testServer, *[]observed) {
	t.Helper()
	ts := &testServer{ping: &pingService{}, store: &storeService{}}

	mod := testmodule.NewModule()
	mod.SetPingService(ts.ping)
	mod.SetEchoService(&echoService{})
	mod.SetComputeService(&computeService{})
	mod.SetStoreService(ts.store)

	var got []observed
	srv := rrpc.NewServer()
	srv.Observe(func(r *http.Request, info rrpc.ResponseInfo) {
		got = append(got, observed{method: r.Method, info: info})
	})
	if err := srv.RegisterModule(mod); err != nil {
		t.Fatal(err)
	}
	ts.handler = srv
	return ts, &got
}

func TestObserver_ReportsRouteTemplateStatusAndSize(t *testing.T) {
	ts, got := newObservedServer(t)

	w := ts.do(t, "GET", "/test/greet/alice", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if len(*got) != 1 {
		t.Fatalf("expected 1 observation, got %d", len(*got))
	}
	o := (*got)[0]
	if o.info.Route != "/test/greet/{name:string}" {
		t.Errorf("Route = %q, want the path template, not the raw path", o.info.Route)
	}
	if o.info.Status != http.StatusOK {
		t.Errorf("Status = %d, want 200", o.info.Status)
	}
	if o.info.Bytes != int64(w.Body.Len()) || o.info.Bytes == 0 {
		t.Errorf("Bytes = %d, want the response body size %d", o.info.Bytes, w.Body.Len())
	}
	if o.info.Duration <= 0 {
		t.Errorf("Duration = %v, want > 0", o.info.Duration)
	}
}

func TestObserver_SeesErrorResponses(t *testing.T) {
	ts, got := newObservedServer(t)
	ts.ping.failWith = rrpc.ErrRrpcBadRequest.WithCause(errors.New("nope"))

	ts.do(t, "GET", "/test/fail", "", "")
	ts.do(t, "GET", "/no/such/path", "", "")
	ts.do(t, "POST", "/test/ping", "application/json", "{}") // wrong method

	if len(*got) != 3 {
		t.Fatalf("expected 3 observations, got %d", len(*got))
	}
	// A handler error: matched route, status from the error body written after the handler returned.
	if o := (*got)[0]; o.info.Route != "/test/fail" || o.info.Status != http.StatusBadRequest || o.info.Bytes == 0 {
		t.Errorf("handler error observed as %+v", o.info)
	}
	// No route: empty template, so unknown paths can't blow up a label's cardinality.
	if o := (*got)[1]; o.info.Route != "" || o.info.Status != http.StatusNotFound {
		t.Errorf("unmatched path observed as %+v", o.info)
	}
	if o := (*got)[2]; o.info.Route != "" || o.info.Status != http.StatusNotFound {
		t.Errorf("wrong method observed as %+v", o.info)
	}
}

func TestObserver_NoObserversLeavesTheWriterAlone(t *testing.T) {
	ts := newTestServer(t)
	w := httptest.NewRecorder()
	ts.handler.ServeHTTP(w, httptest.NewRequest("GET", "/test/ping", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
