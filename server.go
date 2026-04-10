package rrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
)

const ServiceNameCtxKey = "rrpc-service"
const MethodNameCtxKey = "rrpc-method"

var DataMethods = []string{"POST", "PUT"}

type MethodHandlerFunc = func(ctx *Context) error

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
		modules: make([]ServerModule, 0),
	}
}

type ServerModule interface {
	Name() string
	Services() map[string]ServiceHandler
}

type Server struct {
	modules   []ServerModule
	onRequest func(w http.ResponseWriter, r *http.Request) error
	onError   func(r *http.Request, err *RRPCError)
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

	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer s.recoverPanic(w, r)

	ctx := r.Context()
	r = r.WithContext(ctx)

	if s.onRequest != nil {
		if err := s.onRequest(w, r); err != nil {
			rpcErr, ok := err.(RRPCError)
			if !ok {
				rpcErr = ErrRrpcEndpoint.WithCause(err)
			}
			s.sendErrorJSON(w, r, rpcErr)
			return
		}
	}

	err := s.handle(ctx, w, r)

	if err != nil {
		rpcErr, ok := err.(RRPCError)
		if !ok {
			rpcErr = ErrRrpcEndpoint.WithCause(err)
		}
		s.sendErrorJSON(w, r, rpcErr)
	}

}

func (s *Server) handle(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

	requestMethod := r.Method
	requestUri := r.URL.Path

	var methodHandler MethodHandlerFunc
	var params = NewParams()

	found := false

	for _, module := range s.modules {

		for _, serviceHandler := range module.Services() {

			serviceUri := serviceHandler.Path().Regex
			if !strings.HasPrefix(serviceUri, "/") {
				serviceUri = "/" + serviceUri
			}
			if !strings.HasPrefix(requestUri, serviceUri) {
				continue
			}

			for _, method := range serviceHandler.Methods() {

				if requestMethod != method.Type {
					continue
				}

				methodUri := serviceUri
				if !strings.HasSuffix(methodUri, "/") && !strings.HasPrefix(method.Path.Regex, "/") {
					methodUri += "/"
				}
				methodUri += method.Path.Regex

				pattern, err := regexp.Compile(methodUri)
				if err != nil {
					return err
				}

				if !pattern.MatchString(requestUri) {
					continue
				}

				submatch := pattern.FindStringSubmatch(requestUri)
				for i, paramName := range pattern.SubexpNames() {
					if i != 0 && paramName != "" {

						var param PathParam
						if paramFound, ok := serviceHandler.Path().Params[paramName]; ok {
							param = paramFound
						} else if paramFound, ok := method.Path.Params[paramName]; ok {
							param = paramFound
						}

						paramStrValue := submatch[i]
						err = processParam(params, param, paramStrValue)
						if err != nil {
							return err
						}
					}
				}

				if slices.Contains(DataMethods, r.Method) {
					contentType := r.Header.Get(ContentTypeHeader)
					if i := strings.Index(contentType, ";"); i >= 0 {
						contentType = contentType[:i]
					}
					contentType = strings.TrimSpace(strings.ToLower(contentType))

					if contentType != method.ContentType {
						err := ErrRrpcBadRequest.WithCausef("unsupported Content-Type %q (only %s is allowed)", r.Header.Get(ContentTypeHeader), method.ContentType)
						s.sendErrorJSON(w, r, err)
					}
				}

				methodHandler = method.HandlerFunc
				found = true
				break
			}

			if found {
				break
			}

		}

		if found {
			break
		}

	}

	if !found {
		return ErrRrpcBadRoute.WithCausef("no rrpc method defined for path %v", r.URL.Path)
	}

	rrpcCtx := s.createContext(ctx, w, r, params)

	return methodHandler(rrpcCtx)
}

func (s *Server) createContext(ctx context.Context, w http.ResponseWriter, r *http.Request, params *Params) *Context {
	return NewContext(r, w, params)
}

func (s *Server) recoverPanic(w http.ResponseWriter, r *http.Request) {
	// In case of a panic, serve a HTTP 500 error and then panic.
	if rr := recover(); rr != nil {
		s.sendErrorJSON(w, r, ErrRrpcServerPanic.WithCausef("%v", rr))
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
