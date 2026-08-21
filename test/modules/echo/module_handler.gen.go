package echo

import (
    "bytes"
    "context"
    "encoding/json"
    "io"
    "net/http"
    "strconv"

    "github.com/netgarden/rrpc"
)

func newEchoServiceHandler(service EchoService) *EchoServiceHandler {

    serviceHandler := &EchoServiceHandler{}
    serviceHandler.name = "Echo"
    serviceHandler.path = rrpc.ParsePath("/echo")
    serviceHandler.methods = []rrpc.MethodHolder{
        {
            Path:        rrpc.ParsePath("Echo"),
            Type:        "GET",
            ContentType: "application/json",
            HandlerFunc: serviceHandler.handleEcho,
        },
        {
            Path:        rrpc.ParsePath("Chat"),
            Type:        "GET",
            ContentType: "application/json",
            HandlerFunc: serviceHandler.handleChat,
        },
        {
            Path:        rrpc.ParsePath("Collect"),
            Type:        "GET",
            ContentType: "application/json",
            HandlerFunc: serviceHandler.handleCollect,
        },
        {
            Path:        rrpc.ParsePath("Tick"),
            Type:        "GET",
            ContentType: "",
            HandlerFunc: serviceHandler.handleTick,
        },
        {
            Path:        rrpc.ParsePath("UploadBytes"),
            Type:        "GET",
            ContentType: "application/octet-stream",
            HandlerFunc: serviceHandler.handleUploadBytes,
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

func (h *EchoServiceHandler) handleChat(ctx *rrpc.Context) error {
    stream, err := rrpc.UpgradeStream[ChatMessage, ChatReply](ctx, rrpc.UpgradeOptions{})
    if err != nil {
        return err
    }
    return h.service.Chat(ctx, stream)
}

func (h *EchoServiceHandler) handleCollect(ctx *rrpc.Context) error {
    stream, err := rrpc.UpgradeInStream[LogLine](ctx, rrpc.UpgradeOptions{})
    if err != nil {
        return err
    }
    return h.service.Collect(ctx, stream)
}

func (h *EchoServiceHandler) handleTick(ctx *rrpc.Context) error {
    stream, err := rrpc.UpgradeOutStream[TickResponse](ctx, rrpc.UpgradeOptions{})
    if err != nil {
        return err
    }
    return h.service.Tick(ctx, stream)
}

func (h *EchoServiceHandler) handleUploadBytes(ctx *rrpc.Context) error {
    stream, err := rrpc.UpgradeRawInStream(ctx, rrpc.UpgradeOptions{})
    if err != nil {
        return err
    }
    return h.service.UploadBytes(ctx, stream)
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
        path: "/echo",
    }
}

type EchoServiceClientHandler struct {
    name string
    path string
    client *Client
}

func (h *EchoServiceClientHandler) Echo(ctx context.Context, data *EchoRequest) (*EchoResponse, error) {

    methodType := "GET"
    methodPath := "Echo"
    url := joinURLPath(h.client.url, h.path, methodPath)

    jsonBody, err := json.Marshal(data)
    if err != nil {
        return nil, rrpc.ErrRrpcBadRequest.WithCausef("failed to marshal request: %w", err)
    }
    r, err := h.doHttpRequest(ctx, methodType, url, "application/json",bytes.NewBuffer(jsonBody))
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

func (h *EchoServiceClientHandler) Chat(ctx context.Context) (*rrpc.ClientStream[ChatMessage, ChatReply], error) {

    methodPath := "Chat"
    url := joinURLPath(toWebSocketURL(h.client.url), h.path, methodPath)

    return rrpc.DialStream[ChatMessage, ChatReply](ctx, url, rrpc.DialOptions{})
}

func (h *EchoServiceClientHandler) Collect(ctx context.Context) (*rrpc.ClientSendStream[LogLine], error) {

    methodPath := "Collect"
    url := joinURLPath(toWebSocketURL(h.client.url), h.path, methodPath)

    return rrpc.DialSendStream[LogLine](ctx, url, rrpc.DialOptions{})
}

func (h *EchoServiceClientHandler) Tick(ctx context.Context) (*rrpc.ClientRecvStream[TickResponse], error) {

    methodPath := "Tick"
    url := joinURLPath(toWebSocketURL(h.client.url), h.path, methodPath)

    return rrpc.DialRecvStream[TickResponse](ctx, url, rrpc.DialOptions{})
}

func (h *EchoServiceClientHandler) UploadBytes(ctx context.Context) (*rrpc.ClientRawSendStream, error) {

    methodPath := "UploadBytes"
    url := joinURLPath(toWebSocketURL(h.client.url), h.path, methodPath)

    return rrpc.DialRawSendStream(ctx, url, rrpc.DialOptions{})
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
