package test

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strconv"

    "github.com/netgarden/rrpc"
)

func newPingerServiceHandler(service PingerService) *PingerServiceHandler {

    serviceHandler := &PingerServiceHandler{}
    serviceHandler.name = "Pinger"
    serviceHandler.path = rrpc.ParsePath("Pinger")
    serviceHandler.methods = []rrpc.MethodHolder{
        {
            Path:        rrpc.ParsePath("Ping"),
            Type:        "GET",
            ContentType: "",
            HandlerFunc: serviceHandler.handlePing,
        },
    }
    serviceHandler.service = service

    return serviceHandler
}

type PingerServiceHandler struct {
    name    string
    path    *rrpc.Path
    methods []rrpc.MethodHolder

    service PingerService
}

func (h *PingerServiceHandler) Name() string {
    return h.name
}

func (h *PingerServiceHandler) Path() *rrpc.Path {
    return h.path
}

func (h *PingerServiceHandler) Methods() []rrpc.MethodHolder {
    return h.methods
}

func (h *PingerServiceHandler) handlePing(ctx *rrpc.Context) error {

	// Call service method implementation.
    err1 := h.service.Ping(ctx)
    if err1 != nil {
    	rpcErr, ok := err1.(rrpc.RRPCError)
    	if !ok {
    		rpcErr = rrpc.ErrRrpcEndpoint.WithCause(err1)
    	}
    	return rpcErr
    }

	// ctx.Response().WriteHeader(http.StatusOK)

    

	return nil
}



func (h *PingerServiceHandler) readBody(r *http.Request) ([]byte, error) {
    defer r.Body.Close()

    reqBody, err := io.ReadAll(r.Body)
	if err != nil {
	    return nil, rrpc.ErrRrpcBadRequest.WithCausef("failed to read request data: %w", err)
	}

    return reqBody, nil
}
func newPingerServiceClientHandler(client *Client) *PingerServiceClientHandler {
    return &PingerServiceClientHandler{
        client: client,
        name: "Pinger",
        path: "Pinger",
    }
}

type PingerServiceClientHandler struct {
    name string
    path string
    client *Client
}

func (h *PingerServiceClientHandler) Ping(ctx context.Context)error {

    methodType := "GET"
    methodPath := "Ping"
    url := h.client.url + "/" + h.path + "/" + methodPath

    _, err := h.doHttpRequest(ctx, methodType, url, "",nil)
    if err != nil {
        return err
    }

    return nil
}



func (h *PingerServiceClientHandler) doHttpRequest(ctx context.Context, method string, url string, contentType string, body io.Reader) (*SizedReadCloser, error) {

    req, err := http.NewRequest(method, url, body)
    if err != nil {
       	return nil, rrpc.ErrRrpcBadRequest.WithCausef("failed to create http request: %w", err)
    }

    if ctx != nil {
        req = req.WithContext(ctx)
    }

    if body != nil {
        req.Header.Set("Content-Type", contentType)
    }

    req.Header.Set("Accept", "application/json")

    resp, err := h.client.Do(req)
    if err != nil {
        return nil, rrpc.ErrRrpcRequestFailed.WithCause(err)
    }

    if resp.StatusCode != 200 {
        defer resp.Body.Close()

        respBody, err := h.readAll(resp.Body)
        if err != nil {
            return nil, err
        }

        var rpcErr rrpc.RRPCError
		if err := json.Unmarshal(respBody, &rpcErr); err != nil {
			return nil, rrpc.ErrRrpcBadResponse.WithCausef("failed to unmarshal server error: %w", err)
		}

		return nil, rpcErr.WithCauseString()
    }

    var ret *SizedReadCloser

    contentLengthStr := resp.Header.Get("Content-Length")
    if contentLengthStr != "" {

        contentLength, err := strconv.ParseInt(contentLengthStr, 10, 64)
        if err != nil {
            return nil, rrpc.ErrRrpcBadResponse.WithCausef("failed to parse Content-Length header: %w", err)
        }

        ret = NewSizedReadCloser(resp.Body, contentLength)
    }

    return ret, nil
}


func (h *PingerServiceClientHandler) readAll(r io.Reader) ([]byte, error) {
    b, err := io.ReadAll(r)
    if err != nil {
        return nil, rrpc.ErrRrpcBadResponse.WithCausef("failed to read server response body: %w", err)
    }
    return b, nil
}
func newStaticServiceHandler(service StaticService) *StaticServiceHandler {

    serviceHandler := &StaticServiceHandler{}
    serviceHandler.name = "Static"
    serviceHandler.path = rrpc.ParsePath("/")
    serviceHandler.methods = []rrpc.MethodHolder{
        {
            Path:        rrpc.ParsePath("/{filepath:path}"),
            Type:        "GET",
            ContentType: "",
            HandlerFunc: serviceHandler.handleServeStatic,
        },
    }
    serviceHandler.service = service

    return serviceHandler
}

type StaticServiceHandler struct {
    name    string
    path    *rrpc.Path
    methods []rrpc.MethodHolder

    service StaticService
}

func (h *StaticServiceHandler) Name() string {
    return h.name
}

func (h *StaticServiceHandler) Path() *rrpc.Path {
    return h.path
}

func (h *StaticServiceHandler) Methods() []rrpc.MethodHolder {
    return h.methods
}

func (h *StaticServiceHandler) handleServeStatic(ctx *rrpc.Context) error {

	w1 := rrpc.NewResponseWriter(ctx.Response(), "application/octet-stream")

    // Call service method implementation.
    err1 := h.service.ServeStatic(ctx, w1)
    if err1 != nil {
    	rpcErr, ok := err1.(rrpc.RRPCError)
    	if !ok {
    		rpcErr = rrpc.ErrRrpcEndpoint.WithCause(err1)
    	}
    	return rpcErr
    }

	// ctx.Response().WriteHeader(http.StatusOK)

    
    
        if ctx.Response().Header().Get("Content-Type") == "" {
            ctx.Response().Header().Set("Content-Type", "application/octet-stream")
        }

	return nil
}



func (h *StaticServiceHandler) readBody(r *http.Request) ([]byte, error) {
    defer r.Body.Close()

    reqBody, err := io.ReadAll(r.Body)
	if err != nil {
	    return nil, rrpc.ErrRrpcBadRequest.WithCausef("failed to read request data: %w", err)
	}

    return reqBody, nil
}
func newStaticServiceClientHandler(client *Client) *StaticServiceClientHandler {
    return &StaticServiceClientHandler{
        client: client,
        name: "Static",
        path: "/",
    }
}

type StaticServiceClientHandler struct {
    name string
    path string
    client *Client
}

func (h *StaticServiceClientHandler) ServeStatic(ctx context.Context, filepath string) (*SizedReadCloser, error) {

    methodType := "GET"
    methodPath := fmt.Sprintf("/%s", rrpc.EscapePathValue(filepath))
    url := h.client.url + "/" + h.path + "/" + methodPath

    r, err := h.doHttpRequest(ctx, methodType, url, "",nil)
    if err != nil {
        return nil, err
    }

    return r, nil
}



func (h *StaticServiceClientHandler) doHttpRequest(ctx context.Context, method string, url string, contentType string, body io.Reader) (*SizedReadCloser, error) {

    req, err := http.NewRequest(method, url, body)
    if err != nil {
       	return nil, rrpc.ErrRrpcBadRequest.WithCausef("failed to create http request: %w", err)
    }

    if ctx != nil {
        req = req.WithContext(ctx)
    }

    if body != nil {
        req.Header.Set("Content-Type", contentType)
    }

    req.Header.Set("Accept", "application/json")

    resp, err := h.client.Do(req)
    if err != nil {
        return nil, rrpc.ErrRrpcRequestFailed.WithCause(err)
    }

    if resp.StatusCode != 200 {
        defer resp.Body.Close()

        respBody, err := h.readAll(resp.Body)
        if err != nil {
            return nil, err
        }

        var rpcErr rrpc.RRPCError
		if err := json.Unmarshal(respBody, &rpcErr); err != nil {
			return nil, rrpc.ErrRrpcBadResponse.WithCausef("failed to unmarshal server error: %w", err)
		}

		return nil, rpcErr.WithCauseString()
    }

    var ret *SizedReadCloser

    contentLengthStr := resp.Header.Get("Content-Length")
    if contentLengthStr != "" {

        contentLength, err := strconv.ParseInt(contentLengthStr, 10, 64)
        if err != nil {
            return nil, rrpc.ErrRrpcBadResponse.WithCausef("failed to parse Content-Length header: %w", err)
        }

        ret = NewSizedReadCloser(resp.Body, contentLength)
    }

    return ret, nil
}


func (h *StaticServiceClientHandler) readAll(r io.Reader) ([]byte, error) {
    b, err := io.ReadAll(r)
    if err != nil {
        return nil, rrpc.ErrRrpcBadResponse.WithCausef("failed to read server response body: %w", err)
    }
    return b, nil
}
