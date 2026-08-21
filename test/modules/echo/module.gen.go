package echo

import (
    "github.com/netgarden/rrpc"
)

// -- Structs ---------------------------------------------

type ChatMessage struct {
    Text string `json:"text"`
}

type ChatReply struct {
    From string `json:"from"`
    Text string `json:"text"`
}

type EchoRequest struct {
    Message string `json:"message"`
    Count int `json:"count"`
    Flag bool `json:"flag"`
}

type EchoResponse struct {
    Result string `json:"result"`
}

type LogLine struct {
    Line string `json:"line"`
}

type TickResponse struct {
    N int `json:"n"`
}

// -- Services --------------------------------------------

type EchoService interface {
    Echo(ctx *rrpc.Context, request *EchoRequest) (*EchoResponse, error)
    Chat(ctx *rrpc.Context, stream *rrpc.Stream[ChatMessage, ChatReply]) error
    Collect(ctx *rrpc.Context, stream *rrpc.InStream[LogLine]) error
    Tick(ctx *rrpc.Context, stream *rrpc.OutStream[TickResponse]) error
    UploadBytes(ctx *rrpc.Context, stream *rrpc.RawInStream) error
}

// -- Errors ----------------------------------------------


var (
    ErrStreamTriggerError  = rrpc.RRPCError{Code: 1, Name: "StreamTriggerError", Message: "trigger error for testing SendError", HTTPStatus: 400}
)


