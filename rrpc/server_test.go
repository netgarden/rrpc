package rrpc_test

import (
	"bytes"
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
}

func (s *storeService) Upload(ctx *rrpc.Context, r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	s.lastUpload = data
	return nil
}

func (s *storeService) Download(ctx *rrpc.Context, w io.Writer) error {
	_, err := w.Write(s.downloadData)
	return err
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