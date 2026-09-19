package testmodule

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "mime/multipart"
    "net/http"
    "net/url"
    "strconv"

    "github.com/netgarden/rrpc"
)

func newComputeServiceHandler(service ComputeService) *ComputeServiceHandler {

    serviceHandler := &ComputeServiceHandler{}
    serviceHandler.name = "Compute"
    serviceHandler.path = rrpc.ParsePath("/test")
    serviceHandler.methods = []rrpc.MethodHolder{
        {
            Path:        rrpc.ParsePath("/sum"),
            Type:        "POST",
            ContentType: "application/json",
            HandlerFunc: serviceHandler.handleSum,
        },
    }
    serviceHandler.service = service

    return serviceHandler
}

type ComputeServiceHandler struct {
    name    string
    path    *rrpc.Path
    methods []rrpc.MethodHolder

    service ComputeService
}

func (h *ComputeServiceHandler) Name() string {
    return h.name
}

func (h *ComputeServiceHandler) Path() *rrpc.Path {
    return h.path
}

func (h *ComputeServiceHandler) Methods() []rrpc.MethodHolder {
    return h.methods
}

func (h *ComputeServiceHandler) handleSum(ctx *rrpc.Context) error {

	reqPayload := &SumRequest{}
	reqBody, err := h.readBody(ctx.Request())
	if err != nil {
		return err
	}
	if err := json.Unmarshal(reqBody, reqPayload); err != nil {
		return rrpc.ErrRrpcBadRequest.WithCausef("failed to unmarshal request data: %w", err)
	}
	// Call service method implementation.
    ret0, err1 := h.service.Sum(ctx, reqPayload)
    if err1 != nil {
    	rpcErr, ok := err1.(rrpc.RRPCError)
    	if !ok {
    		rpcErr = rrpc.ErrRrpcEndpoint.WithCause(err1)
    	}
    	return rpcErr
    }

	// ctx.Response().WriteHeader(http.StatusOK)

    
    
	respBody, err := json.Marshal(ret0)
	if err != nil {
		return rrpc.ErrRrpcBadResponse.WithCausef("failed to marshal json response: %w", err)
	}

    ctx.Response().Header().Set("Content-Type", "application/json")
    ctx.Response().Write(respBody)

	

	return nil
}



func (h *ComputeServiceHandler) readBody(r *http.Request) ([]byte, error) {
    defer r.Body.Close()

    reqBody, err := io.ReadAll(r.Body)
	if err != nil {
	    return nil, rrpc.ErrRrpcBadRequest.WithCausef("failed to read request data: %w", err)
	}

    return reqBody, nil
}
func newComputeServiceClientHandler(client *Client) *ComputeServiceClientHandler {
    return &ComputeServiceClientHandler{
        client: client,
        name: "Compute",
        path: "/test",
    }
}

type ComputeServiceClientHandler struct {
    name string
    path string
    client *Client
}

func (h *ComputeServiceClientHandler) Sum(ctx context.Context, data *SumRequest) (*SumResponse, error) {

    methodType := "POST"
    methodPath := "/sum"
    url := joinURLPath(h.client.url, h.path, methodPath)

    jsonBody, err := json.Marshal(data)
    if err != nil {
        return nil, rrpc.ErrRrpcBadRequest.WithCausef("failed to marshal request: %w", err)
    }
    r, err := h.doHttpRequest(ctx, methodType, url, "application/json",bytes.NewBuffer(jsonBody))
    if err != nil {
        return nil, err
    }

    out := &SumResponse{}
    defer r.Close()
    respBody, err := h.readAll(r)
    if err != nil {
        return nil, err
    }
    if err := json.Unmarshal(respBody, out); err != nil {
        return nil, rrpc.ErrRrpcBadResponse.WithCausef("failed to unmarshal response: %w", err)
    }

    return out, nil
}



func (h *ComputeServiceClientHandler) doHttpRequest(ctx context.Context, method string, url string, contentType string, body io.Reader) (*SizedReadCloser, error) {

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


func (h *ComputeServiceClientHandler) readAll(r io.Reader) ([]byte, error) {
    b, err := io.ReadAll(r)
    if err != nil {
        return nil, rrpc.ErrRrpcBadResponse.WithCausef("failed to read server response body: %w", err)
    }
    return b, nil
}
func newEchoServiceHandler(service EchoService) *EchoServiceHandler {

    serviceHandler := &EchoServiceHandler{}
    serviceHandler.name = "Echo"
    serviceHandler.path = rrpc.ParsePath("/test")
    serviceHandler.methods = []rrpc.MethodHolder{
        {
            Path:        rrpc.ParsePath("/echo"),
            Type:        "GET",
            ContentType: "application/json",
            HandlerFunc: serviceHandler.handleEcho,
        },
    }
    serviceHandler.service = service

    return serviceHandler
}

type EchoServiceHandler struct {
    name    string
    path    *rrpc.Path
    methods []rrpc.MethodHolder

    service EchoService
}

func (h *EchoServiceHandler) Name() string {
    return h.name
}

func (h *EchoServiceHandler) Path() *rrpc.Path {
    return h.path
}

func (h *EchoServiceHandler) Methods() []rrpc.MethodHolder {
    return h.methods
}

func (h *EchoServiceHandler) handleEcho(ctx *rrpc.Context) error {

	reqPayload := &EchoRequest{}
	if err := rrpc.DecodeQueryParams(ctx.Request().URL.Query(), reqPayload); err != nil {
		return rrpc.ErrRrpcBadRequest.WithCausef("failed to decode query params: %w", err)
	}
	// Call service method implementation.
    ret0, err1 := h.service.Echo(ctx, reqPayload)
    if err1 != nil {
    	rpcErr, ok := err1.(rrpc.RRPCError)
    	if !ok {
    		rpcErr = rrpc.ErrRrpcEndpoint.WithCause(err1)
    	}
    	return rpcErr
    }

	// ctx.Response().WriteHeader(http.StatusOK)

    
    
	respBody, err := json.Marshal(ret0)
	if err != nil {
		return rrpc.ErrRrpcBadResponse.WithCausef("failed to marshal json response: %w", err)
	}

    ctx.Response().Header().Set("Content-Type", "application/json")
    ctx.Response().Write(respBody)

	

	return nil
}



func (h *EchoServiceHandler) readBody(r *http.Request) ([]byte, error) {
    defer r.Body.Close()

    reqBody, err := io.ReadAll(r.Body)
	if err != nil {
	    return nil, rrpc.ErrRrpcBadRequest.WithCausef("failed to read request data: %w", err)
	}

    return reqBody, nil
}
func newEchoServiceClientHandler(client *Client) *EchoServiceClientHandler {
    return &EchoServiceClientHandler{
        client: client,
        name: "Echo",
        path: "/test",
    }
}

type EchoServiceClientHandler struct {
    name string
    path string
    client *Client
}

func (h *EchoServiceClientHandler) Echo(ctx context.Context, data *EchoRequest) (*EchoResponse, error) {

    methodType := "GET"
    methodPath := "/echo"
    url := joinURLPath(h.client.url, h.path, methodPath)

    query, err := rrpc.EncodeQueryParams(data)
    if err != nil {
        return nil, rrpc.ErrRrpcBadRequest.WithCausef("failed to encode query params: %w", err)
    }
    if len(query) > 0 {
        url += "?" + query.Encode()
    }
    r, err := h.doHttpRequest(ctx, methodType, url, "application/json",nil)
    if err != nil {
        return nil, err
    }

    out := &EchoResponse{}
    defer r.Close()
    respBody, err := h.readAll(r)
    if err != nil {
        return nil, err
    }
    if err := json.Unmarshal(respBody, out); err != nil {
        return nil, rrpc.ErrRrpcBadResponse.WithCausef("failed to unmarshal response: %w", err)
    }

    return out, nil
}



func (h *EchoServiceClientHandler) doHttpRequest(ctx context.Context, method string, url string, contentType string, body io.Reader) (*SizedReadCloser, error) {

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


func (h *EchoServiceClientHandler) readAll(r io.Reader) ([]byte, error) {
    b, err := io.ReadAll(r)
    if err != nil {
        return nil, rrpc.ErrRrpcBadResponse.WithCausef("failed to read server response body: %w", err)
    }
    return b, nil
}
func newPingServiceHandler(service PingService) *PingServiceHandler {

    serviceHandler := &PingServiceHandler{}
    serviceHandler.name = "Ping"
    serviceHandler.path = rrpc.ParsePath("/test")
    serviceHandler.methods = []rrpc.MethodHolder{
        {
            Path:        rrpc.ParsePath("/ping"),
            Type:        "GET",
            ContentType: "",
            HandlerFunc: serviceHandler.handlePing,
        },
        {
            Path:        rrpc.ParsePath("/greet/{name:string}"),
            Type:        "GET",
            ContentType: "",
            HandlerFunc: serviceHandler.handleGreet,
        },
        {
            Path:        rrpc.ParsePath("/fail"),
            Type:        "GET",
            ContentType: "",
            HandlerFunc: serviceHandler.handleFail,
        },
    }
    serviceHandler.service = service

    return serviceHandler
}

type PingServiceHandler struct {
    name    string
    path    *rrpc.Path
    methods []rrpc.MethodHolder

    service PingService
}

func (h *PingServiceHandler) Name() string {
    return h.name
}

func (h *PingServiceHandler) Path() *rrpc.Path {
    return h.path
}

func (h *PingServiceHandler) Methods() []rrpc.MethodHolder {
    return h.methods
}

func (h *PingServiceHandler) handlePing(ctx *rrpc.Context) error {

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

func (h *PingServiceHandler) handleGreet(ctx *rrpc.Context) error {

	// Call service method implementation.
    ret0, err1 := h.service.Greet(ctx)
    if err1 != nil {
    	rpcErr, ok := err1.(rrpc.RRPCError)
    	if !ok {
    		rpcErr = rrpc.ErrRrpcEndpoint.WithCause(err1)
    	}
    	return rpcErr
    }

	// ctx.Response().WriteHeader(http.StatusOK)

    
    
	respBody, err := json.Marshal(ret0)
	if err != nil {
		return rrpc.ErrRrpcBadResponse.WithCausef("failed to marshal json response: %w", err)
	}

    ctx.Response().Header().Set("Content-Type", "application/json")
    ctx.Response().Write(respBody)

	

	return nil
}

func (h *PingServiceHandler) handleFail(ctx *rrpc.Context) error {

	// Call service method implementation.
    err1 := h.service.Fail(ctx)
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



func (h *PingServiceHandler) readBody(r *http.Request) ([]byte, error) {
    defer r.Body.Close()

    reqBody, err := io.ReadAll(r.Body)
	if err != nil {
	    return nil, rrpc.ErrRrpcBadRequest.WithCausef("failed to read request data: %w", err)
	}

    return reqBody, nil
}
func newPingServiceClientHandler(client *Client) *PingServiceClientHandler {
    return &PingServiceClientHandler{
        client: client,
        name: "Ping",
        path: "/test",
    }
}

type PingServiceClientHandler struct {
    name string
    path string
    client *Client
}

func (h *PingServiceClientHandler) Ping(ctx context.Context)error {

    methodType := "GET"
    methodPath := "/ping"
    url := joinURLPath(h.client.url, h.path, methodPath)

    _, err := h.doHttpRequest(ctx, methodType, url, "",nil)
    if err != nil {
        return err
    }

    return nil
}

func (h *PingServiceClientHandler) Greet(ctx context.Context, name string) (*GreetResponse, error) {

    methodType := "GET"
    methodPath := fmt.Sprintf("/greet/%s", url.PathEscape(name))
    url := joinURLPath(h.client.url, h.path, methodPath)

    r, err := h.doHttpRequest(ctx, methodType, url, "",nil)
    if err != nil {
        return nil, err
    }

    out := &GreetResponse{}
    defer r.Close()
    respBody, err := h.readAll(r)
    if err != nil {
        return nil, err
    }
    if err := json.Unmarshal(respBody, out); err != nil {
        return nil, rrpc.ErrRrpcBadResponse.WithCausef("failed to unmarshal response: %w", err)
    }

    return out, nil
}

func (h *PingServiceClientHandler) Fail(ctx context.Context)error {

    methodType := "GET"
    methodPath := "/fail"
    url := joinURLPath(h.client.url, h.path, methodPath)

    _, err := h.doHttpRequest(ctx, methodType, url, "",nil)
    if err != nil {
        return err
    }

    return nil
}



func (h *PingServiceClientHandler) doHttpRequest(ctx context.Context, method string, url string, contentType string, body io.Reader) (*SizedReadCloser, error) {

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


func (h *PingServiceClientHandler) readAll(r io.Reader) ([]byte, error) {
    b, err := io.ReadAll(r)
    if err != nil {
        return nil, rrpc.ErrRrpcBadResponse.WithCausef("failed to read server response body: %w", err)
    }
    return b, nil
}
func newStoreServiceHandler(service StoreService) *StoreServiceHandler {

    serviceHandler := &StoreServiceHandler{}
    serviceHandler.name = "Store"
    serviceHandler.path = rrpc.ParsePath("/test")
    serviceHandler.methods = []rrpc.MethodHolder{
        {
            Path:        rrpc.ParsePath("/upload"),
            Type:        "PUT",
            ContentType: "application/octet-stream",
            HandlerFunc: serviceHandler.handleUpload,
        },
        {
            Path:        rrpc.ParsePath("/upload2"),
            Type:        "POST",
            ContentType: "multipart/form-data",
            HandlerFunc: serviceHandler.handleUpload2,
        },
        {
            Path:        rrpc.ParsePath("/download"),
            Type:        "GET",
            ContentType: "",
            HandlerFunc: serviceHandler.handleDownload,
        },
    }
    serviceHandler.service = service

    return serviceHandler
}

type StoreServiceHandler struct {
    name    string
    path    *rrpc.Path
    methods []rrpc.MethodHolder

    service StoreService
}

func (h *StoreServiceHandler) Name() string {
    return h.name
}

func (h *StoreServiceHandler) Path() *rrpc.Path {
    return h.path
}

func (h *StoreServiceHandler) Methods() []rrpc.MethodHolder {
    return h.methods
}

func (h *StoreServiceHandler) handleUpload(ctx *rrpc.Context) error {

	// Call service method implementation.
    err1 := h.service.Upload(ctx, ctx.Request().Body)
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

func (h *StoreServiceHandler) handleUpload2(ctx *rrpc.Context) error {

	if err := ctx.Request().ParseMultipartForm(32 << 20); err != nil {
		return rrpc.ErrRrpcBadRequest.WithCausef("failed to parse multipart form: %w", err)
	}
	reqPayload := &UploadInfo{}
	if err := json.Unmarshal([]byte(ctx.Request().FormValue("info")), reqPayload); err != nil {
		return rrpc.ErrRrpcBadRequest.WithCausef("failed to unmarshal info field: %w", err)
	}
	dataFile, _, err := ctx.Request().FormFile("data")
	if err != nil {
		return rrpc.ErrRrpcBadRequest.WithCausef("failed to read data field: %w", err)
	}
	defer dataFile.Close()

	// Call service method implementation.
    err1 := h.service.Upload2(ctx, reqPayload, dataFile)
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

func (h *StoreServiceHandler) handleDownload(ctx *rrpc.Context) error {

	w1 := rrpc.NewResponseWriter(ctx.Response(), "application/octet-stream")

    // Call service method implementation.
    err1 := h.service.Download(ctx, w1)
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



func (h *StoreServiceHandler) readBody(r *http.Request) ([]byte, error) {
    defer r.Body.Close()

    reqBody, err := io.ReadAll(r.Body)
	if err != nil {
	    return nil, rrpc.ErrRrpcBadRequest.WithCausef("failed to read request data: %w", err)
	}

    return reqBody, nil
}
func newStoreServiceClientHandler(client *Client) *StoreServiceClientHandler {
    return &StoreServiceClientHandler{
        client: client,
        name: "Store",
        path: "/test",
    }
}

type StoreServiceClientHandler struct {
    name string
    path string
    client *Client
}

func (h *StoreServiceClientHandler) Upload(ctx context.Context, r0 io.Reader)error {

    methodType := "PUT"
    methodPath := "/upload"
    url := joinURLPath(h.client.url, h.path, methodPath)

    _, err := h.doHttpRequest(ctx, methodType, url, "application/octet-stream",r0)
    if err != nil {
        return err
    }

    return nil
}

func (h *StoreServiceClientHandler) Upload2(ctx context.Context, data *UploadInfo, r0 io.Reader)error {

    methodType := "POST"
    methodPath := "/upload2"
    url := joinURLPath(h.client.url, h.path, methodPath)

    _, err := h.doMultipartRequest(ctx, methodType, url, "info", data, "data", r0)
    if err != nil {
        return err
    }

    return nil
}

func (h *StoreServiceClientHandler) Download(ctx context.Context) (*SizedReadCloser, error) {

    methodType := "GET"
    methodPath := "/download"
    url := joinURLPath(h.client.url, h.path, methodPath)

    r, err := h.doHttpRequest(ctx, methodType, url, "",nil)
    if err != nil {
        return nil, err
    }

    return r, nil
}



func (h *StoreServiceClientHandler) doHttpRequest(ctx context.Context, method string, url string, contentType string, body io.Reader) (*SizedReadCloser, error) {

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

func (h *StoreServiceClientHandler) doMultipartRequest(ctx context.Context, method string, url string, structField string, in interface{}, bytesField string, r io.Reader) (*SizedReadCloser, error) {

    var buf bytes.Buffer
    mw := multipart.NewWriter(&buf)

    jsonData, err := json.Marshal(in)
    if err != nil {
        return nil, rrpc.ErrRrpcBadRequest.WithCausef("failed to marshal json part: %w", err)
    }
    fw, err := mw.CreateFormField(structField)
    if err != nil {
        return nil, rrpc.ErrRrpcBadRequest.WithCausef("failed to create form field: %w", err)
    }
    if _, err := fw.Write(jsonData); err != nil {
        return nil, rrpc.ErrRrpcBadRequest.WithCausef("failed to write form field: %w", err)
    }

    bw, err := mw.CreateFormFile(bytesField, bytesField)
    if err != nil {
        return nil, rrpc.ErrRrpcBadRequest.WithCausef("failed to create form file: %w", err)
    }
    if _, err := io.Copy(bw, r); err != nil {
        return nil, rrpc.ErrRrpcBadRequest.WithCausef("failed to write bytes part: %w", err)
    }
    mw.Close()

    req, err := http.NewRequest(method, url, &buf)
    if err != nil {
        return nil, rrpc.ErrRrpcBadRequest.WithCausef("failed to create http request: %w", err)
    }
    if ctx != nil {
        req = req.WithContext(ctx)
    }
    req.Header.Set("Content-Type", mw.FormDataContentType())
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

func (h *StoreServiceClientHandler) readAll(r io.Reader) ([]byte, error) {
    b, err := io.ReadAll(r)
    if err != nil {
        return nil, rrpc.ErrRrpcBadResponse.WithCausef("failed to read server response body: %w", err)
    }
    return b, nil
}
