package files

import (
    "bytes"
    "context"
    "encoding/json"
    "errors"
    "io"
    "net/http"
    "strconv"

    "github.com/netgarden/rrpc"
)

func newFilesServiceHandler(service FilesService) *FilesServiceHandler {

    serviceHandler := &FilesServiceHandler{}
    serviceHandler.name = "Files"
    serviceHandler.path = rrpc.ParsePath("/files")
    serviceHandler.methods = []rrpc.MethodHolder{
        {
            Path:        rrpc.ParsePath("/upload/{filename:string}"),
            Type:        "POST",
            ContentType: "application/octet-stream",
            HandlerFunc: serviceHandler.handleUpload,
        },
        {
            Path:        rrpc.ParsePath("/download/{filename:string}"),
            Type:        "GET",
            ContentType: "",
            HandlerFunc: serviceHandler.handleDownload,
        },
    }
    serviceHandler.service = service

    return serviceHandler
}

type FilesServiceHandler struct {
    name    string
    path    *rrpc.Path
    methods []rrpc.MethodHolder

    service FilesService
}

func (h *FilesServiceHandler) Name() string {
    return h.name
}

func (h *FilesServiceHandler) Path() *rrpc.Path {
    return h.path
}

func (h *FilesServiceHandler) Methods() []rrpc.MethodHolder {
    return h.methods
}

func (h *FilesServiceHandler) handleUpload(ctx *rrpc.Context) error {

	// Call service method implementation.
    err1 := h.service.Upload(ctx, ctx.Request().Body)
    if err1 != nil {
    	rpcErr, ok := err1.(RRPCError)
    	if !ok {
    		rpcErr = ErrRrpcEndpoint.WithCause(err1)
    	}
    	return rpcErr
    }

	// ctx.Response().WriteHeader(http.StatusOK)

    

	return nil
}

func (h *FilesServiceHandler) handleDownload(ctx *rrpc.Context) error {

	w1 := rrpc.NewResponseWriter(ctx.Response(), "application/octet-stream")

    // Call service method implementation.
    err1 := h.service.Download(ctx, w1)
    if err1 != nil {
    	rpcErr, ok := err1.(RRPCError)
    	if !ok {
    		rpcErr = ErrRrpcEndpoint.WithCause(err1)
    	}
    	return rpcErr
    }

	// ctx.Response().WriteHeader(http.StatusOK)

    
    
        ctx.Response().Header().Set("Content-Type", "application/octet-stream")

	return nil
}



func (h *FilesServiceHandler) readBody(r *http.Request) ([]byte, error) {
    defer r.Body.Close()

    reqBody, err := io.ReadAll(r.Body)
	if err != nil {
	    return nil, ErrRrpcBadRequest.WithCausef("failed to read request data: %w", err)
	}

    return reqBody, nil
}

func newFilesServiceClientHandler(client *Client) *FilesServiceClientHandler {
    return &FilesServiceClientHandler{
        client: client,
        name: "Files",
        path: "/files",
    }
}

type FilesServiceClientHandler struct {
    name string
    path string
    client *Client
}

func (h *FilesServiceClientHandler) Upload(ctx context.Context, r0 *SizedReader)error {

    methodType := "POST"
    methodPath := "/upload/{filename:string}"
    methodContentType := "application/octet-stream"
    url := h.client.url + "/" + h.path + "/" + methodPath

    _, err := h.doHttpRequest(ctx, methodType, url, methodContentType,r0)
    if err != nil {
        return err
    }

    return nil
}

func (h *FilesServiceClientHandler) Download(ctx context.Context) (*SizedReadCloser, error) {

    methodType := "GET"
    methodPath := "/download/{filename:string}"
    methodContentType := ""
    url := h.client.url + "/" + h.path + "/" + methodPath

    r, err := h.doHttpRequest(ctx, methodType, url, methodContentType,nil)
    if err != nil {
        return nil, err
    }

    return r, nil
}



func (h *FilesServiceClientHandler) doHttpRequest(ctx context.Context, method string, url string, contentType string, in interface{}) (*SizedReadCloser, error) {

    var reqBodyBuffer io.Reader = nil

    if in != nil {

        reqBody, err := json.Marshal(in)
        if err != nil {
            return nil, ErrRrpcBadRequest.WithCausef("failed to marshal json request: %w", err)
        }

        reqBodyBuffer = bytes.NewBuffer(reqBody)
    }

    req, err := http.NewRequest(method, url, reqBodyBuffer)
    if err != nil {
       	return nil, ErrRrpcBadRequest.WithCausef("failed to create http request: %w", err)
    }

    if ctx != nil {
        req = req.WithContext(ctx)
    }

    if in != nil {
        req.Header.Set("Content-Type", contentType)
    }

    req.Header.Set("Accept", "application/json")

    resp, err := h.client.Do(req)
    if err != nil {
        return nil, ErrRrpcRequestFailed.WithCause(err)
    }

    if resp.StatusCode != 200 {
        defer resp.Body.Close()

        respBody, err := h.readAll(resp.Body)
        if err != nil {
            return nil, err
        }

        var rpcErr RRPCError
		if err := json.Unmarshal(respBody, &rpcErr); err != nil {
			return nil, ErrRrpcBadResponse.WithCausef("failed to unmarshal server error: %w", err)
		}
		if rpcErr.Cause != "" {
			rpcErr.cause = errors.New(rpcErr.Cause)
		}

		return nil, rpcErr
    }

    var ret *SizedReadCloser

    contentLengthStr := resp.Header.Get("Content-Length")
    if contentLengthStr != "" {

        contentLength, err := strconv.ParseInt(contentLengthStr, 10, 64)
        if err != nil {
            return nil, ErrRrpcBadResponse.WithCausef("failed to parse Content-Length header: %w", err)
        }

        ret = NewSizedReadCloser(resp.Body, contentLength)
    }

    return ret, nil
}

func (h *FilesServiceClientHandler) readAll(r io.Reader) ([]byte, error) {
    b, err := io.ReadAll(r)
    if err != nil {
        return nil, ErrRrpcBadResponse.WithCausef("failed to read server response body: %w", err)
    }
    return b, nil
}
