package files

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

func newFilesServiceHandler(service FilesService) *FilesServiceHandler {

    serviceHandler := &FilesServiceHandler{}
    serviceHandler.name = "Files"
    serviceHandler.path = rrpc.ParsePath("/files")
    serviceHandler.methods = []rrpc.MethodHolder{
        {
            Path:        rrpc.ParsePath("/upload/{filename:string}"),
            Type:        "PUT",
            ContentType: "application/octet-stream",
            HandlerFunc: serviceHandler.handleUpload,
        },
        {
            Path:        rrpc.ParsePath("/uploadJSON/{filename:string}"),
            Type:        "PUT",
            ContentType: "application/json",
            HandlerFunc: serviceHandler.handleUploadJSON,
        },
        {
            Path:        rrpc.ParsePath("/upload2"),
            Type:        "POST",
            ContentType: "multipart/form-data",
            HandlerFunc: serviceHandler.handleUpload2,
        },
        {
            Path:        rrpc.ParsePath("/upload2PDF"),
            Type:        "POST",
            ContentType: "multipart/form-data",
            HandlerFunc: serviceHandler.handleUpload2PDF,
        },
        {
            Path:        rrpc.ParsePath("/download/{filename:string}"),
            Type:        "GET",
            ContentType: "",
            HandlerFunc: serviceHandler.handleDownload,
        },
        {
            Path:        rrpc.ParsePath("/downloadPDF/{filename:string}"),
            Type:        "GET",
            ContentType: "",
            HandlerFunc: serviceHandler.handleDownloadPDF,
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
    	rpcErr, ok := err1.(rrpc.RRPCError)
    	if !ok {
    		rpcErr = rrpc.ErrRrpcEndpoint.WithCause(err1)
    	}
    	return rpcErr
    }

	// ctx.Response().WriteHeader(http.StatusOK)

    

	return nil
}

func (h *FilesServiceHandler) handleUploadJSON(ctx *rrpc.Context) error {

	// Call service method implementation.
    err1 := h.service.UploadJSON(ctx, ctx.Request().Body)
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

func (h *FilesServiceHandler) handleUpload2(ctx *rrpc.Context) error {

	if err := ctx.Request().ParseMultipartForm(32 << 20); err != nil {
		return rrpc.ErrRrpcBadRequest.WithCausef("failed to parse multipart form: %w", err)
	}
	reqPayload := &UploadDataInfo{}
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

func (h *FilesServiceHandler) handleUpload2PDF(ctx *rrpc.Context) error {

	if err := ctx.Request().ParseMultipartForm(32 << 20); err != nil {
		return rrpc.ErrRrpcBadRequest.WithCausef("failed to parse multipart form: %w", err)
	}
	reqPayload := &UploadDataInfo{}
	if err := json.Unmarshal([]byte(ctx.Request().FormValue("info")), reqPayload); err != nil {
		return rrpc.ErrRrpcBadRequest.WithCausef("failed to unmarshal info field: %w", err)
	}
	dataFile, _, err := ctx.Request().FormFile("data")
	if err != nil {
		return rrpc.ErrRrpcBadRequest.WithCausef("failed to read data field: %w", err)
	}
	defer dataFile.Close()

	// Call service method implementation.
    err1 := h.service.Upload2PDF(ctx, reqPayload, dataFile)
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

func (h *FilesServiceHandler) handleDownload(ctx *rrpc.Context) error {

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

func (h *FilesServiceHandler) handleDownloadPDF(ctx *rrpc.Context) error {

	w1 := rrpc.NewResponseWriter(ctx.Response(), "application/pdf")

    // Call service method implementation.
    err1 := h.service.DownloadPDF(ctx, w1)
    if err1 != nil {
    	rpcErr, ok := err1.(rrpc.RRPCError)
    	if !ok {
    		rpcErr = rrpc.ErrRrpcEndpoint.WithCause(err1)
    	}
    	return rpcErr
    }

	// ctx.Response().WriteHeader(http.StatusOK)

    
    
        if ctx.Response().Header().Get("Content-Type") == "" {
            ctx.Response().Header().Set("Content-Type", "application/pdf")
        }

	return nil
}



func (h *FilesServiceHandler) readBody(r *http.Request) ([]byte, error) {
    defer r.Body.Close()

    reqBody, err := io.ReadAll(r.Body)
	if err != nil {
	    return nil, rrpc.ErrRrpcBadRequest.WithCausef("failed to read request data: %w", err)
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

func (h *FilesServiceClientHandler) Upload(ctx context.Context, filename string, r0 io.Reader)error {

    methodType := "PUT"
    methodPath := fmt.Sprintf("/upload/%s", url.PathEscape(filename))
    url := joinURLPath(h.client.url, h.path, methodPath)

    _, err := h.doHttpRequest(ctx, methodType, url, "application/octet-stream",r0)
    if err != nil {
        return err
    }

    return nil
}

func (h *FilesServiceClientHandler) UploadJSON(ctx context.Context, filename string, r0 io.Reader)error {

    methodType := "PUT"
    methodPath := fmt.Sprintf("/uploadJSON/%s", url.PathEscape(filename))
    url := joinURLPath(h.client.url, h.path, methodPath)

    _, err := h.doHttpRequest(ctx, methodType, url, "application/json",r0)
    if err != nil {
        return err
    }

    return nil
}

func (h *FilesServiceClientHandler) Upload2(ctx context.Context, data *UploadDataInfo, r0 io.Reader)error {

    methodType := "POST"
    methodPath := "/upload2"
    url := joinURLPath(h.client.url, h.path, methodPath)

    _, err := h.doMultipartRequest(ctx, methodType, url, "info", data, "data", r0, "application/octet-stream")
    if err != nil {
        return err
    }

    return nil
}

func (h *FilesServiceClientHandler) Upload2PDF(ctx context.Context, data *UploadDataInfo, r0 io.Reader)error {

    methodType := "POST"
    methodPath := "/upload2PDF"
    url := joinURLPath(h.client.url, h.path, methodPath)

    _, err := h.doMultipartRequest(ctx, methodType, url, "info", data, "data", r0, "application/pdf")
    if err != nil {
        return err
    }

    return nil
}

func (h *FilesServiceClientHandler) Download(ctx context.Context, filename string) (*SizedReadCloser, error) {

    methodType := "GET"
    methodPath := fmt.Sprintf("/download/%s", url.PathEscape(filename))
    url := joinURLPath(h.client.url, h.path, methodPath)

    r, err := h.doHttpRequest(ctx, methodType, url, "",nil)
    if err != nil {
        return nil, err
    }

    return r, nil
}

func (h *FilesServiceClientHandler) DownloadPDF(ctx context.Context, filename string) (*SizedReadCloser, error) {

    methodType := "GET"
    methodPath := fmt.Sprintf("/downloadPDF/%s", url.PathEscape(filename))
    url := joinURLPath(h.client.url, h.path, methodPath)

    r, err := h.doHttpRequest(ctx, methodType, url, "",nil)
    if err != nil {
        return nil, err
    }

    return r, nil
}



func (h *FilesServiceClientHandler) doHttpRequest(ctx context.Context, method string, url string, contentType string, body io.Reader) (*SizedReadCloser, error) {

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

func (h *FilesServiceClientHandler) doMultipartRequest(ctx context.Context, method string, url string, structField string, in interface{}, bytesField string, r io.Reader, bytesContentType string) (*SizedReadCloser, error) {

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

    bw, err := mw.CreatePart(rrpc.FormDataPartHeader(bytesField, bytesField, bytesContentType))
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

func (h *FilesServiceClientHandler) readAll(r io.Reader) ([]byte, error) {
    b, err := io.ReadAll(r)
    if err != nil {
        return nil, rrpc.ErrRrpcBadResponse.WithCausef("failed to read server response body: %w", err)
    }
    return b, nil
}
