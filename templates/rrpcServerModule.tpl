package {{ if index .Options "go_package" }}{{ .Options.go_package }}{{ else }}rpc{{ end }}

import (
    "context"
    "encoding/json"
    "net/http"
    "slices"
    "strings"
)

const ServiceNameCtxKey = "rrpc-service"
const MethodNameCtxKey = "rrpc-method"

type MethodHandlerFunc = func(ctx context.Context, w http.ResponseWriter, r *http.Request) error

type ServiceHandler interface {
    Name() string
    Path() string
    Methods() []MethodHolder
}

type MethodHolder struct {
    Path        string
    Type        string
    ContentType string
    HandlerFunc MethodHandlerFunc
}

func NewServerModule() *Server {
    return &ServerModule{
        Services: make([]ServiceHandler, 0),
    }
}

type ServerModule struct {
    Name() string
    Services() []ServiceHandler
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

    var servicePath string
    var serviceHandler ServiceHandler
    var methodHandler MethodHandlerFunc

    found := false

    out:
    for servicePath, serviceHandler = range s.serviceHandlers {
        for _, method := range serviceHandler.Methods() {

            methodUri := "/" + servicePath + "/" + method.Path
            if requestUri != methodUri {
                continue
            }

            if requestMethod != method.Type {
                continue
            }

            if slices.Contains(DataMethods, r.Method) {
                contentType := r.Header.Get("Content-Type")
                if i := strings.Index(contentType, ";"); i >= 0 {
                    contentType = contentType[:i]
                }
                contentType = strings.TrimSpace(strings.ToLower(contentType))

                if contentType != method.ContentType {
                    err := ErrRrpcBadRequest.WithCausef("unsupported Content-Type %q (only %s is allowed)", r.Header.Get("Content-Type"), method.ContentType)
                    s.sendErrorJSON(w, r, err)
                }
            }

            methodHandler = method.HandlerFunc
            found = true
            break out
        }
    }

    if !found {
        return ErrRrpcBadRoute.WithCausef("no rrpc method defined for path %v", r.URL.Path)
    }

    ctx = context.WithValue(ctx, ServiceNameCtxKey, serviceHandler.Name())

    return methodHandler(ctx, w, r)
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(rpcErr.HTTPStatus)

	respBody, _ := json.Marshal(rpcErr)
	w.Write(respBody)
}

func (s *Server) OnRequest(f func(w http.ResponseWriter, r *http.Request) error) {
    s.onRequest = f
}

func (s *Server) OnError(f func(r *http.Request, err *RRPCError)) {
    s.onError = f
}

{{ range $_, $service := .Services -}}
func (s *Server) Set{{ $service.Name }}Service({{ firstLetterToLower $service.Name }}Service {{ $service.Name }}Service) {
    s.serviceHandlers["{{ $service.Name }}"] = new{{ firstLetterToUpper $service.Name }}ServiceHandler({{ firstLetterToLower $service.Name }}Service)
}
{{- end }}

func NewResponseWriter(w http.ResponseWriter, contentType string) *ResponseWriter {
    return &ResponseWriter{
        w: w,
        contentType: contentType,
    }
}

type ResponseWriter struct {
    w              http.ResponseWriter
    contentType    string
    contentTypeSet bool
}

func (w *ResponseWriter) Write(b []byte) (int, error) {

    if !w.contentTypeSet && w.w.Header().Get("Content-Type") != "" {
        w.w.Header().Set("Content-Type", w.contentType)
    }
    w.contentTypeSet = true

    return w.w.Write(b)
}
