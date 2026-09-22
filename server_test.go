package rrpc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/netgarden/rrpc"
	"github.com/netgarden/rrpc/internal/testmodule"
)

// ---------------------------------------------------------------------------
// Service implementations
// ---------------------------------------------------------------------------

type pingService struct {
	failWith error
}

func (s *pingService) Ping(ctx *rrpc.Context) error { return nil }

func (s *pingService) Greet(ctx *rrpc.Context) (*testmodule.GreetResponse, error) {
	name := ctx.Params().GetString("name")
	return &testmodule.GreetResponse{Greeting: "Hello, " + name + "!"}, nil
}

func (s *pingService) Fail(ctx *rrpc.Context) error { return s.failWith }

type echoService struct{}

func (s *echoService) Echo(ctx *rrpc.Context, req *testmodule.EchoRequest) (*testmodule.EchoResponse, error) {
	result := fmt.Sprintf("%s x%d", req.Message, req.Count)
	return &testmodule.EchoResponse{Result: result}, nil
}

type computeService struct{}

func (s *computeService) Sum(ctx *rrpc.Context, req *testmodule.SumRequest) (*testmodule.SumResponse, error) {
	return &testmodule.SumResponse{Total: req.A + req.B}, nil
}

type storeService struct {
	lastUpload     []byte
	lastUploadName string
	downloadData   []byte
	// downloadContentTypeOverride, when set, is written by Download/DownloadPDF
	// before the first Write - exercises that a handler can still override the
	// response Content-Type at runtime, same as before @ResponseContentType.
	downloadContentTypeOverride string
}

func (s *storeService) Upload(ctx *rrpc.Context, r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	s.lastUpload = data
	return nil
}

func (s *storeService) UploadJSON(ctx *rrpc.Context, r io.Reader) error {
	return s.Upload(ctx, r)
}

func (s *storeService) Download(ctx *rrpc.Context, w io.Writer) error {
	if s.downloadContentTypeOverride != "" {
		ctx.Response().Header().Set("Content-Type", s.downloadContentTypeOverride)
	}
	_, err := w.Write(s.downloadData)
	return err
}

func (s *storeService) DownloadPDF(ctx *rrpc.Context, w io.Writer) error {
	return s.Download(ctx, w)
}

func (s *storeService) Upload2(ctx *rrpc.Context, info *testmodule.UploadInfo, r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	s.lastUpload = data
	s.lastUploadName = info.Filename
	return nil
}

func (s *storeService) Upload2PDF(ctx *rrpc.Context, info *testmodule.UploadInfo, r io.Reader) error {
	return s.Upload2(ctx, info, r)
}

// ---------------------------------------------------------------------------
// Test server setup
// ---------------------------------------------------------------------------

type testServer struct {
	ping    *pingService
	store   *storeService
	handler http.Handler
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()

	ts := &testServer{
		ping:  &pingService{},
		store: &storeService{},
	}

	mod := testmodule.NewModule()
	mod.SetPingService(ts.ping)
	mod.SetEchoService(&echoService{})
	mod.SetComputeService(&computeService{})
	mod.SetStoreService(ts.store)

	srv := rrpc.NewServer()
	if err := srv.RegisterModule(mod); err != nil {
		t.Fatal(err)
	}
	ts.handler = srv

	return ts
}

func (ts *testServer) do(t *testing.T, method, path, contentType, body string) *httptest.ResponseRecorder {
	t.Helper()
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, bodyReader)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	w := httptest.NewRecorder()
	ts.handler.ServeHTTP(w, req)
	return w
}

func readRRPCError(t *testing.T, w *httptest.ResponseRecorder) rrpc.RRPCError {
	t.Helper()
	var e rrpc.RRPCError
	if err := json.Unmarshal(w.Body.Bytes(), &e); err != nil {
		t.Fatalf("failed to parse error body: %v — body: %s", err, w.Body.String())
	}
	return e
}

// ---------------------------------------------------------------------------
// Routing
// ---------------------------------------------------------------------------

func TestServer_RouteNotFound(t *testing.T) {
	ts := newTestServer(t)
	w := ts.do(t, "GET", "/no/such/path", "", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	e := readRRPCError(t, w)
	if e.Code != rrpc.ErrRrpcBadRoute.Code {
		t.Errorf("expected ErrRrpcBadRoute code %d, got %d", rrpc.ErrRrpcBadRoute.Code, e.Code)
	}
}

func TestServer_WrongMethod_ReturnsNotFound(t *testing.T) {
	ts := newTestServer(t)
	// /test/ping is GET-only; POST should return bad route
	w := ts.do(t, "POST", "/test/ping", "application/json", "{}")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestServer_WrongContentType(t *testing.T) {
	ts := newTestServer(t)
	w := ts.do(t, "POST", "/test/sum", "text/plain", `{"a":1,"b":2}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// TestServer_WrongContentType_SingleCleanResponse is the regression test for
// a real bug: the Content-Type mismatch branch used to call sendErrorJSON
// itself and then fall through into running the handler anyway (no return),
// so the response body was the error JSON immediately followed by a second,
// unrelated JSON body - and, for a method with a side effect, the side
// effect still happened despite the 400. Assert both: exactly one
// well-formed JSON error body, and the handler never ran.
func TestServer_WrongContentType_SingleCleanResponse(t *testing.T) {
	ts := newTestServer(t)
	w := ts.do(t, "PUT", "/test/upload", "application/json", "binary content here")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d — body: %s", w.Code, w.Body.String())
	}
	var rpcErr rrpc.RRPCError
	dec := json.NewDecoder(w.Body)
	if err := dec.Decode(&rpcErr); err != nil {
		t.Fatalf("response body is not valid JSON: %v (body: %s)", err, w.Body.String())
	}
	if dec.More() {
		t.Fatalf("response body has trailing data after the error JSON (the double-write bug): %s", w.Body.String())
	}
	if ts.store.lastUpload != nil {
		t.Errorf("handler ran despite the Content-Type mismatch: lastUpload = %q", ts.store.lastUpload)
	}
}

func TestServer_ContentType_WithCharset(t *testing.T) {
	ts := newTestServer(t)
	w := ts.do(t, "POST", "/test/sum", "application/json; charset=utf-8", `{"a":3,"b":4}`)
	if w.Code == http.StatusBadRequest {
		t.Fatalf("should accept application/json with charset, got 400: %s", w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// GET — no inputs/outputs
// ---------------------------------------------------------------------------

func TestServer_Ping(t *testing.T) {
	ts := newTestServer(t)
	w := ts.do(t, "GET", "/test/ping", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// GET — path params
// ---------------------------------------------------------------------------

func TestServer_PathParam(t *testing.T) {
	ts := newTestServer(t)
	w := ts.do(t, "GET", "/test/greet/Alice", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	var resp testmodule.GreetResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Greeting != "Hello, Alice!" {
		t.Errorf("expected 'Hello, Alice!', got %q", resp.Greeting)
	}
}

// ---------------------------------------------------------------------------
// GET — query params decoded into struct
// ---------------------------------------------------------------------------

func TestServer_EchoQueryParams(t *testing.T) {
	ts := newTestServer(t)
	w := ts.do(t, "GET", "/test/echo?message=hello&count=3", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	var resp testmodule.EchoResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Result != "hello x3" {
		t.Errorf("expected 'hello x3', got %q", resp.Result)
	}
}

// ---------------------------------------------------------------------------
// POST — JSON body decoded into struct
// ---------------------------------------------------------------------------

func TestServer_PostJSON(t *testing.T) {
	ts := newTestServer(t)
	w := ts.do(t, "POST", "/test/sum", "application/json", `{"a":10,"b":32}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	var resp testmodule.SumResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Total != 42 {
		t.Errorf("expected total 42, got %d", resp.Total)
	}
}

// ---------------------------------------------------------------------------
// PUT — raw bytes body
// ---------------------------------------------------------------------------

func TestServer_UploadBytes(t *testing.T) {
	ts := newTestServer(t)
	payload := []byte("binary content here")
	req := httptest.NewRequest("PUT", "/test/upload", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/octet-stream")
	w := httptest.NewRecorder()
	ts.handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	if !bytes.Equal(ts.store.lastUpload, payload) {
		t.Errorf("uploaded data mismatch: got %q", ts.store.lastUpload)
	}
}

// TestServer_UploadJSON_OverriddenContentType exercises
// @RequestContentType("application/json") on uploadJSON: the annotation
// changes what the *server* requires, not just what a generated client sends
// - so the old default (application/octet-stream) must now be rejected for
// this method, and the declared value accepted.
func TestServer_UploadJSON_OverriddenContentType(t *testing.T) {
	ts := newTestServer(t)
	payload := []byte(`{"not":"actually parsed, just embedded"}`)

	w := ts.do(t, "PUT", "/test/uploadJSON", "application/json", string(payload))
	if w.Code != http.StatusOK {
		t.Fatalf("application/json should be accepted: expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	if !bytes.Equal(ts.store.lastUpload, payload) {
		t.Errorf("uploaded data mismatch: got %q", ts.store.lastUpload)
	}

	w = ts.do(t, "PUT", "/test/uploadJSON", "application/octet-stream", "irrelevant")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("the old default should now be rejected for this method: expected 400, got %d — body: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// POST — multipart (struct + bytes)
// ---------------------------------------------------------------------------

func TestServer_UploadMultipart(t *testing.T) {
	ts := newTestServer(t)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	infoJSON, _ := json.Marshal(testmodule.UploadInfo{Filename: "test.bin"})
	fw, _ := mw.CreateFormField("info")
	fw.Write(infoJSON)

	bw, _ := mw.CreateFormFile("data", "data")
	bw.Write([]byte("multipart payload"))

	mw.Close()

	req := httptest.NewRequest("POST", "/test/upload2", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	ts.handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	if ts.store.lastUploadName != "test.bin" {
		t.Errorf("expected filename 'test.bin', got %q", ts.store.lastUploadName)
	}
	if string(ts.store.lastUpload) != "multipart payload" {
		t.Errorf("expected payload 'multipart payload', got %q", ts.store.lastUpload)
	}
}

// TestClient_MultipartBytesPartContentType is the case the whole envelope/
// part split exists for: @RequestContentType on a multipart method
// (upload2PDF) must change the *bytes field's own part* Content-Type, not
// the envelope - which must keep carrying the boundary untouched, exactly
// like an unannotated multipart method (upload2). Exercised through the
// real generated client (not a hand-built request), against a plain capture
// handler rather than the full rrpc dispatch, since the server intentionally
// doesn't expose a multipart part's own header back to the service method
// (see the "Multipart server-side validation" decision - this test is about
// what the client sends, not server-side enforcement).
func TestClient_MultipartBytesPartContentType(t *testing.T) {
	cases := []struct {
		name       string
		call       func(c *testmodule.Client) error
		wantPartCT string
	}{
		{
			name: "unannotated upload2 keeps the old default",
			call: func(c *testmodule.Client) error {
				return c.Store.Upload2(context.Background(), &testmodule.UploadInfo{Filename: "x.bin"}, strings.NewReader("data"))
			},
			wantPartCT: "application/octet-stream",
		},
		{
			name: "upload2PDF uses the @RequestContentType override",
			call: func(c *testmodule.Client) error {
				return c.Store.Upload2PDF(context.Background(), &testmodule.UploadInfo{Filename: "x.pdf"}, strings.NewReader("data"))
			},
			wantPartCT: "application/pdf",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotPartCT, gotEnvelopeCT string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotEnvelopeCT = r.Header.Get("Content-Type")
				if err := r.ParseMultipartForm(1 << 20); err != nil {
					t.Fatal(err)
				}
				if fhs := r.MultipartForm.File["data"]; len(fhs) > 0 {
					gotPartCT = fhs[0].Header.Get("Content-Type")
				}
				w.Header().Set("Content-Length", "0")
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			client := testmodule.NewClient(srv.URL)
			if err := tc.call(client); err != nil {
				t.Fatal(err)
			}
			if gotPartCT != tc.wantPartCT {
				t.Errorf("multipart part Content-Type = %q, want %q", gotPartCT, tc.wantPartCT)
			}
			if !strings.HasPrefix(gotEnvelopeCT, "multipart/form-data; boundary=") {
				t.Errorf("envelope Content-Type = %q, want multipart/form-data with a boundary - unaffected by the annotation", gotEnvelopeCT)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GET — bytes response
// ---------------------------------------------------------------------------

func TestServer_DownloadBytes(t *testing.T) {
	ts := newTestServer(t)
	ts.store.downloadData = []byte("hello bytes")

	w := ts.do(t, "GET", "/test/download", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/octet-stream" {
		t.Errorf("expected Content-Type application/octet-stream, got %q", ct)
	}
	if !bytes.Equal(w.Body.Bytes(), ts.store.downloadData) {
		t.Errorf("expected body %q, got %q", ts.store.downloadData, w.Body.Bytes())
	}
}

// TestServer_DownloadPDF_OverriddenContentType exercises
// @ResponseContentType("application/pdf") on downloadPDF: the default the
// server sets (when the handler hasn't already set one) follows the
// annotation instead of the old hardcoded application/octet-stream.
func TestServer_DownloadPDF_OverriddenContentType(t *testing.T) {
	ts := newTestServer(t)
	ts.store.downloadData = []byte("%PDF-1.4 pretend this is a pdf")

	w := ts.do(t, "GET", "/test/downloadPDF", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/pdf" {
		t.Errorf("expected Content-Type application/pdf, got %q", ct)
	}
	if !bytes.Equal(w.Body.Bytes(), ts.store.downloadData) {
		t.Errorf("expected body %q, got %q", ts.store.downloadData, w.Body.Bytes())
	}
}

// TestServer_DownloadPDF_HandlerCanStillOverrideAtRuntime confirms the
// "only if not already set" escape hatch (response_writer.go) still works
// when the default itself now comes from @ResponseContentType: a handler
// that sets its own Content-Type before writing still wins.
func TestServer_DownloadPDF_HandlerCanStillOverrideAtRuntime(t *testing.T) {
	ts := newTestServer(t)
	ts.store.downloadData = []byte("data")
	ts.store.downloadContentTypeOverride = "application/x-custom"

	w := ts.do(t, "GET", "/test/downloadPDF", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/x-custom" {
		t.Errorf("handler-set Content-Type should win over @ResponseContentType's default: got %q", ct)
	}
}

// ---------------------------------------------------------------------------
// Error propagation
// ---------------------------------------------------------------------------

func TestServer_HandlerReturns_RRPCError(t *testing.T) {
	ts := newTestServer(t)
	ts.ping.failWith = testmodule.ErrNotFoundError

	w := ts.do(t, "GET", "/test/fail", "", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	e := readRRPCError(t, w)
	if e.Code != testmodule.ErrNotFoundError.Code {
		t.Errorf("expected ErrNotFoundError code %d, got %d", testmodule.ErrNotFoundError.Code, e.Code)
	}
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/problem+json") {
		t.Errorf("expected Content-Type application/problem+json, got %q", ct)
	}
}

func TestServer_HandlerReturns_GenericError(t *testing.T) {
	ts := newTestServer(t)
	ts.ping.failWith = fmt.Errorf("something unexpected")

	w := ts.do(t, "GET", "/test/fail", "", "")
	if w.Code != rrpc.ErrRrpcEndpoint.HTTPStatus {
		t.Fatalf("expected %d, got %d", rrpc.ErrRrpcEndpoint.HTTPStatus, w.Code)
	}
	e := readRRPCError(t, w)
	if e.Code != rrpc.ErrRrpcEndpoint.Code {
		t.Errorf("expected ErrRrpcEndpoint code %d, got %d", rrpc.ErrRrpcEndpoint.Code, e.Code)
	}
}

func TestServer_HandlerReturns_ForbiddenError(t *testing.T) {
	ts := newTestServer(t)
	ts.ping.failWith = testmodule.ErrForbiddenError

	w := ts.do(t, "GET", "/test/fail", "", "")
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
	e := readRRPCError(t, w)
	if e.Code != testmodule.ErrForbiddenError.Code {
		t.Errorf("expected ErrForbiddenError code %d, got %d", testmodule.ErrForbiddenError.Code, e.Code)
	}
}

// ---------------------------------------------------------------------------
// onRequest hook
// ---------------------------------------------------------------------------

func TestServer_OnRequestHook_Blocks(t *testing.T) {
	mod := testmodule.NewModule()
	mod.SetPingService(&pingService{})

	srv := rrpc.NewServer()
	srv.RegisterModule(mod)
	srv.SetOnRequest(func(w http.ResponseWriter, r *http.Request) error {
		return rrpc.RRPCError{Code: 99, Name: "Unauthorized", Message: "not allowed", HTTPStatus: http.StatusUnauthorized}
	})

	req := httptest.NewRequest("GET", "/test/ping", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestServer_OnRequestHook_Passes(t *testing.T) {
	mod := testmodule.NewModule()
	mod.SetPingService(&pingService{})

	srv := rrpc.NewServer()
	srv.RegisterModule(mod)
	srv.SetOnRequest(func(w http.ResponseWriter, r *http.Request) error {
		return nil // allow
	})

	req := httptest.NewRequest("GET", "/test/ping", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Middleware
// ---------------------------------------------------------------------------

func TestServer_Middleware_Executes(t *testing.T) {
	mod := testmodule.NewModule()
	mod.SetPingService(&pingService{})

	var called bool
	srv := rrpc.NewServer()
	srv.RegisterModule(mod)
	srv.Use(func(ctx *rrpc.Context, next func(*rrpc.Context) error) error {
		called = true
		return next(ctx)
	})

	req := httptest.NewRequest("GET", "/test/ping", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if !called {
		t.Error("middleware was not called")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestServer_Middleware_Order(t *testing.T) {
	mod := testmodule.NewModule()
	mod.SetPingService(&pingService{})

	var order []int
	srv := rrpc.NewServer()
	srv.RegisterModule(mod)
	srv.Use(
		func(ctx *rrpc.Context, next func(*rrpc.Context) error) error {
			order = append(order, 1)
			err := next(ctx)
			order = append(order, 4)
			return err
		},
		func(ctx *rrpc.Context, next func(*rrpc.Context) error) error {
			order = append(order, 2)
			err := next(ctx)
			order = append(order, 3)
			return err
		},
	)

	req := httptest.NewRequest("GET", "/test/ping", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	expected := []int{1, 2, 3, 4}
	for i, v := range expected {
		if i >= len(order) || order[i] != v {
			t.Errorf("expected middleware order %v, got %v", expected, order)
			break
		}
	}
}

func TestServer_Middleware_ShortCircuit(t *testing.T) {
	mod := testmodule.NewModule()
	mod.SetPingService(&pingService{})

	srv := rrpc.NewServer()
	srv.RegisterModule(mod)
	srv.Use(func(ctx *rrpc.Context, next func(*rrpc.Context) error) error {
		return rrpc.RRPCError{Code: 99, Name: "Unauthorized", Message: "not allowed", HTTPStatus: http.StatusUnauthorized}
	})

	req := httptest.NewRequest("GET", "/test/ping", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestServer_Middleware_AccessesPathParams(t *testing.T) {
	mod := testmodule.NewModule()
	mod.SetPingService(&pingService{})

	var capturedName string
	srv := rrpc.NewServer()
	srv.RegisterModule(mod)
	srv.Use(func(ctx *rrpc.Context, next func(*rrpc.Context) error) error {
		capturedName = ctx.Params().GetString("name")
		return next(ctx)
	})

	req := httptest.NewRequest("GET", "/test/greet/Alice", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	if capturedName != "Alice" {
		t.Errorf("expected 'Alice', got %q", capturedName)
	}
}

func TestServer_Middleware_AddsResponseHeader(t *testing.T) {
	mod := testmodule.NewModule()
	mod.SetPingService(&pingService{})

	srv := rrpc.NewServer()
	srv.RegisterModule(mod)
	srv.Use(func(ctx *rrpc.Context, next func(*rrpc.Context) error) error {
		ctx.Response().Header().Set("X-Middleware", "yes")
		return next(ctx)
	})

	req := httptest.NewRequest("GET", "/test/ping", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if got := w.Header().Get("X-Middleware"); got != "yes" {
		t.Errorf("expected X-Middleware header 'yes', got %q", got)
	}
}

// ---------------------------------------------------------------------------
// Duplicate module registration
// ---------------------------------------------------------------------------

func TestServer_RegisterModule_Duplicate(t *testing.T) {
	srv := rrpc.NewServer()
	mod := testmodule.NewModule()
	if err := srv.RegisterModule(mod); err != nil {
		t.Fatal(err)
	}
	if err := srv.RegisterModule(mod); err == nil {
		t.Error("expected error when registering same module twice")
	}
}

// ---------------------------------------------------------------------------
// ParsePath
// ---------------------------------------------------------------------------

func TestParsePath_NoParams(t *testing.T) {
	p := rrpc.ParsePath("/api/ping")
	if p.Regex != "/api/ping" {
		t.Errorf("expected regex '/api/ping', got %q", p.Regex)
	}
	if len(p.Params) != 0 {
		t.Errorf("expected no params, got %v", p.Params)
	}
}

func TestParsePath_StringParam(t *testing.T) {
	p := rrpc.ParsePath("/files/{filename:string}")
	if _, ok := p.Params["filename"]; !ok {
		t.Fatal("expected param 'filename'")
	}
	if p.Params["filename"].Type != "string" {
		t.Errorf("expected type 'string', got %q", p.Params["filename"].Type)
	}
	if !strings.Contains(p.Regex, "(?P<filename>") {
		t.Errorf("regex missing named capture group: %s", p.Regex)
	}
}

func TestParsePath_PathParam(t *testing.T) {
	p := rrpc.ParsePath("/static/{filepath:path}")
	if p.Params["filepath"].Type != "path" {
		t.Errorf("expected type 'path', got %q", p.Params["filepath"].Type)
	}
}

func TestParsePath_MultipleParams(t *testing.T) {
	p := rrpc.ParsePath("/a/{x:string}/b/{y:string}")
	if len(p.Params) != 2 {
		t.Errorf("expected 2 params, got %d", len(p.Params))
	}
}

// ---------------------------------------------------------------------------
// DecodeQueryParams
// ---------------------------------------------------------------------------

func TestDecodeQueryParams_Primitives(t *testing.T) {
	type Q struct {
		Name  string  `json:"name"`
		Count int     `json:"count"`
		Score float64 `json:"score"`
		Flag  bool    `json:"flag"`
	}
	values := map[string][]string{
		"name":  {"hello"},
		"count": {"42"},
		"score": {"3.14"},
		"flag":  {"true"},
	}
	var q Q
	if err := rrpc.DecodeQueryParams(values, &q); err != nil {
		t.Fatal(err)
	}
	if q.Name != "hello" {
		t.Errorf("Name: want 'hello', got %q", q.Name)
	}
	if q.Count != 42 {
		t.Errorf("Count: want 42, got %d", q.Count)
	}
	if q.Score != 3.14 {
		t.Errorf("Score: want 3.14, got %f", q.Score)
	}
	if !q.Flag {
		t.Errorf("Flag: want true, got false")
	}
}

func TestDecodeQueryParams_EmptySlice_Skipped(t *testing.T) {
	type Q struct{ Name string `json:"name"` }
	values := map[string][]string{"name": {}}
	var q Q
	if err := rrpc.DecodeQueryParams(values, &q); err != nil {
		t.Fatal(err)
	}
	if q.Name != "" {
		t.Errorf("expected empty name, got %q", q.Name)
	}
}

func TestDecodeQueryParams_MultipleValues_UsesFirst(t *testing.T) {
	type Q struct{ X int `json:"x"` }
	values := map[string][]string{"x": {"1", "2", "3"}}
	var q Q
	if err := rrpc.DecodeQueryParams(values, &q); err != nil {
		t.Fatal(err)
	}
	if q.X != 1 {
		t.Errorf("expected first value 1, got %d", q.X)
	}
}