package rrpc_test

import (
	"net/http/httptest"
	"testing"

	"github.com/netgarden/rrpc"
)

// fakeServiceHandler and fakeServerModule are minimal hand-built test
// doubles — deliberately not generated code — so these tests can register
// a catch-all alongside a specific route in whatever order/grouping each
// test needs, independent of any particular codegen output.

type fakeServiceHandler struct {
	name    string
	path    *rrpc.Path
	methods []rrpc.MethodHolder
}

func (h *fakeServiceHandler) Name() string                 { return h.name }
func (h *fakeServiceHandler) Path() *rrpc.Path             { return h.path }
func (h *fakeServiceHandler) Methods() []rrpc.MethodHolder { return h.methods }

func newGetHandler(name, path, body string) *fakeServiceHandler {
	return &fakeServiceHandler{
		name: name,
		path: rrpc.ParsePath(path),
		methods: []rrpc.MethodHolder{
			{
				Path: rrpc.ParsePath(""),
				Type: "GET",
				HandlerFunc: func(ctx *rrpc.Context) error {
					_, err := ctx.Response().Write([]byte(body))
					return err
				},
			},
		},
	}
}

func newCatchAllHandler(name, body string) *fakeServiceHandler {
	return &fakeServiceHandler{
		name: name,
		path: rrpc.ParsePath(""),
		methods: []rrpc.MethodHolder{
			{
				Path: rrpc.ParsePath("{filepath:path}"),
				Type: "GET",
				HandlerFunc: func(ctx *rrpc.Context) error {
					_, err := ctx.Response().Write([]byte(body))
					return err
				},
			},
		},
	}
}

type fakeServerModule struct {
	name     string
	services map[string]rrpc.ServiceHandler
}

func (m *fakeServerModule) Name() string                             { return m.name }
func (m *fakeServerModule) Services() map[string]rrpc.ServiceHandler { return m.services }

func doGet(t *testing.T, srv *rrpc.Server, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	return w
}

// TestRouting_SpecificBeatsCatchAll_RegardlessOfRegistrationOrder is the
// direct regression test for the bug this file's routing rewrite fixes: a
// specific route must always win over an unrelated catch-all that also
// matches the same request, no matter which was registered first, and no
// matter whether they're registered as separate modules or together under
// one module's Services() map (previously a Go map — iteration order is
// randomized per process, so "first registered" was never actually
// reliable in that case either).
func TestRouting_SpecificBeatsCatchAll_RegardlessOfRegistrationOrder(t *testing.T) {

	cases := []struct {
		name    string
		modules func() []rrpc.ServerModule
	}{
		{
			name: "specific module registered before catch-all module",
			modules: func() []rrpc.ServerModule {
				return []rrpc.ServerModule{
					&fakeServerModule{name: "api", services: map[string]rrpc.ServiceHandler{
						"Servers": newGetHandler("Servers", "api/servers", "specific"),
					}},
					&fakeServerModule{name: "ui", services: map[string]rrpc.ServiceHandler{
						"Static": newCatchAllHandler("Static", "catchall"),
					}},
				}
			},
		},
		{
			name: "catch-all module registered before specific module",
			modules: func() []rrpc.ServerModule {
				return []rrpc.ServerModule{
					&fakeServerModule{name: "ui", services: map[string]rrpc.ServiceHandler{
						"Static": newCatchAllHandler("Static", "catchall"),
					}},
					&fakeServerModule{name: "api", services: map[string]rrpc.ServiceHandler{
						"Servers": newGetHandler("Servers", "api/servers", "specific"),
					}},
				}
			},
		},
		{
			name: "both services registered together under a single module",
			modules: func() []rrpc.ServerModule {
				return []rrpc.ServerModule{
					&fakeServerModule{name: "combined", services: map[string]rrpc.ServiceHandler{
						"Servers": newGetHandler("Servers", "api/servers", "specific"),
						"Static":  newCatchAllHandler("Static", "catchall"),
					}},
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			srv := rrpc.NewServer()
			for _, mod := range tc.modules() {
				if err := srv.RegisterModule(mod); err != nil {
					t.Fatalf("RegisterModule: %v", err)
				}
			}

			if w := doGet(t, srv, "/api/servers"); w.Body.String() != "specific" {
				t.Errorf("GET /api/servers: expected the specific route to win, got body %q", w.Body.String())
			}

			if w := doGet(t, srv, "/index.html"); w.Body.String() != "catchall" {
				t.Errorf("GET /index.html: expected the catch-all route to handle it, got body %q", w.Body.String())
			}

			if w := doGet(t, srv, "/assets/js/app.js"); w.Body.String() != "catchall" {
				t.Errorf("GET /assets/js/app.js: expected the catch-all route to handle a nested path, got body %q", w.Body.String())
			}
		})
	}
}

// TestRouting_StableAcrossManyRegistrationPermutations runs the same
// scenario across a large number of module-registration orderings — since
// Services() is a map, the original bug (whichever service the map
// happened to yield first won) would only surface nondeterministically, so
// a single run isn't a reliable regression guard on its own.
func TestRouting_StableAcrossManyRegistrationPermutations(t *testing.T) {

	for i := 0; i < 50; i++ {

		// Vary the number of unrelated services sharing the module map
		// with the catch-all, to exercise different map sizes/iteration
		// behavior across runs.
		services := map[string]rrpc.ServiceHandler{
			"Static": newCatchAllHandler("Static", "catchall"),
		}
		for j := 0; j < i%5; j++ {
			name := string(rune('A' + j))
			services[name] = newGetHandler(name, "api/"+name, "specific-"+name)
		}
		services["Servers"] = newGetHandler("Servers", "api/servers", "specific")

		srv := rrpc.NewServer()
		if err := srv.RegisterModule(&fakeServerModule{name: "combined", services: services}); err != nil {
			t.Fatalf("RegisterModule: %v", err)
		}

		if w := doGet(t, srv, "/api/servers"); w.Body.String() != "specific" {
			t.Fatalf("iteration %d: expected the specific route to win, got body %q", i, w.Body.String())
		}
	}
}
