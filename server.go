package rrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strings"
)

const ServiceNameCtxKey = "rrpc-service"
const MethodNameCtxKey = "rrpc-method"

var DataMethods = []string{"POST", "PUT"}

type MethodHandlerFunc = func(ctx *Context) error

type Middleware func(ctx *Context, next func(*Context) error) error

type ServiceHandler interface {
	Name() string
	Path() *Path
	Methods() []MethodHolder
}

type MethodHolder struct {
	Path        *Path
	Type        string
	ContentType string
	HandlerFunc MethodHandlerFunc
}

func NewServer() *Server {
	return &Server{
		modules:     make([]ServerModule, 0),
		routes:      make([]route, 0),
		middlewares: make([]Middleware, 0),
	}
}

type ServerModule interface {
	Name() string
	Services() map[string]ServiceHandler
}

type Server struct {
	modules []ServerModule
	// routes is every (service, method) pair across all registered
	// modules, flattened and kept sorted by specificity (see route.go) so
	// routing never depends on ServerModule.Services()' map iteration
	// order — e.g. a catch-all service registered anywhere can never
	// nondeterministically shadow a more specific one.
	routes      []route
	middlewares []Middleware
	onRequest   func(w http.ResponseWriter, r *http.Request) error
	onError     func(r *http.Request, err *RRPCError)
}

func (s *Server) Use(middlewares ...Middleware) {
	s.middlewares = append(s.middlewares, middlewares...)
}

func (s *Server) SetOnRequest(fn func(w http.ResponseWriter, r *http.Request) error) {
	s.onRequest = fn
}

func (s *Server) RegisterModule(module ServerModule) error {

	for _, m := range s.modules {
		if m == module {
			return fmt.Errorf("module %s already registered", module.Name())
		}
	}

	s.modules = append(s.modules, module)

	for _, serviceHandler := range module.Services() {
		for _, method := range serviceHandler.Methods() {
			r, err := newRoute(serviceHandler, method)
			if err != nil {
				return err
			}
			s.routes = append(s.routes, r)
		}
	}

	sort.SliceStable(s.routes, func(i, j int) bool {
		return s.routes[i].specificity.less(s.routes[j].specificity)
	})

	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// rrpcCtx is filled in by coreHandle and read by recoverPanic's defer
	// - a *Context pointer so a panic occurring after a successful
	// WebSocket upgrade (Context.Hijacked() == true) is still known about
	// by the deferred recover, which must not attempt to write a JSON
	// error body to a connection that's no longer a normal ResponseWriter.
	var rrpcCtx *Context
	defer s.recoverPanic(w, r, &rrpcCtx)
	rrpcCtx = s.coreHandle(w, r)
}

func (s *Server) coreHandle(w http.ResponseWriter, r *http.Request) *Context {
	ctx := r.Context()

	if s.onRequest != nil {
		if err := s.onRequest(w, r); err != nil {
			s.sendErrorJSON(w, r, toRRPCError(err))
			return nil
		}
	}

	rrpcCtx, err := s.handle(ctx, w, r)

	if err != nil && (rrpcCtx == nil || !rrpcCtx.Hijacked()) {
		s.sendErrorJSON(w, r, toRRPCError(err))
	}

	return rrpcCtx
}

func (s *Server) handle(ctx context.Context, w http.ResponseWriter, r *http.Request) (*Context, error) {

	requestMethod := r.Method
	requestUri := r.URL.Path

	var matched *route

	// s.routes is kept sorted by specificity (see route.go), so the first
	// match here is always the most specific one, deterministically.
	for i := range s.routes {

		candidate := &s.routes[i]

		if candidate.method.Type != requestMethod {
			continue
		}

		if !candidate.pattern.MatchString(requestUri) {
			continue
		}

		matched = candidate
		break
	}

	if matched == nil {
		return nil, ErrRrpcBadRoute.WithCausef("no rrpc method defined for path %v", r.URL.Path)
	}

	params := NewParams()

	submatch := matched.pattern.FindStringSubmatch(requestUri)
	for i, paramName := range matched.pattern.SubexpNames() {
		if i != 0 && paramName != "" {

			var param PathParam
			if paramFound, ok := matched.serviceHandler.Path().Params[paramName]; ok {
				param = paramFound
			} else if paramFound, ok := matched.method.Path.Params[paramName]; ok {
				param = paramFound
			}

			paramStrValue := submatch[i]
			if err := processParam(params, param, paramStrValue); err != nil {
				return nil, err
			}
		}
	}

	if slices.Contains(DataMethods, r.Method) {
		contentType := r.Header.Get(ContentTypeHeader)
		if i := strings.Index(contentType, ";"); i >= 0 {
			contentType = contentType[:i]
		}
		contentType = strings.TrimSpace(strings.ToLower(contentType))

		if contentType != matched.method.ContentType {
			// Every other error path in this function just returns; coreHandle
			// (the caller) is the one that calls sendErrorJSON. Doing it here
			// too (as this used to) both duplicates that and - since nothing
			// stopped the handler from still running afterward - produced a
			// corrupted, doubly-written response.
			return nil, ErrRrpcBadRequest.WithCausef("unsupported Content-Type %q (only %s is allowed)", r.Header.Get(ContentTypeHeader), matched.method.ContentType)
		}
	}

	methodHandler := matched.method.HandlerFunc
	rrpcCtx := s.createContext(ctx, w, r, params)

	handler := func(c *Context) error {
		return methodHandler(c)
	}
	for i := len(s.middlewares) - 1; i >= 0; i-- {
		next := handler
		mw := s.middlewares[i]
		handler = func(c *Context) error {
			return mw(c, next)
		}
	}

	return rrpcCtx, handler(rrpcCtx)
}

func (s *Server) createContext(ctx context.Context, w http.ResponseWriter, r *http.Request, params *Params) *Context {
	return NewContext(r, w, params)
}

func (s *Server) recoverPanic(w http.ResponseWriter, r *http.Request, rrpcCtx **Context) {
	// In case of a panic, serve a HTTP 500 error and then panic - unless
	// the connection was already hijacked by a WebSocket upgrade, in
	// which case there's no ResponseWriter left to write to.
	if rr := recover(); rr != nil {
		if *rrpcCtx == nil || !(*rrpcCtx).Hijacked() {
			s.sendErrorJSON(w, r, ErrRrpcServerPanic.WithCausef("%v", rr))
		}
		panic(rr)
	}
}

func (s *Server) sendErrorJSON(w http.ResponseWriter, r *http.Request, rpcErr RRPCError) {

	if s.onError != nil {
		s.onError(r, &rpcErr)
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(rpcErr.HTTPStatus)

	respBody, _ := json.Marshal(rpcErr)
	w.Write(respBody)
}
