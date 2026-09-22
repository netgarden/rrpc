package testmodule

import (
    "github.com/netgarden/rrpc"

    "io"
)

// -- Structs ---------------------------------------------

type EchoRequest struct {
    Message string `json:"message"`
    Count int `json:"count"`
}

type EchoResponse struct {
    Result string `json:"result"`
}

type GreetResponse struct {
    Greeting string `json:"greeting"`
}

type SumRequest struct {
    A int `json:"a"`
    B int `json:"b"`
}

type SumResponse struct {
    Total int `json:"total"`
}

type UploadInfo struct {
    Filename string `json:"filename"`
}

// -- Services --------------------------------------------

type ComputeService interface {
    Sum(ctx *rrpc.Context, request *SumRequest) (*SumResponse, error)
}

type EchoService interface {
    Echo(ctx *rrpc.Context, request *EchoRequest) (*EchoResponse, error)
}

type PingService interface {
    Ping(ctx *rrpc.Context) error
    Greet(ctx *rrpc.Context) (*GreetResponse, error)
    Fail(ctx *rrpc.Context) error
}

type StoreService interface {
    Upload(ctx *rrpc.Context, r io.Reader) error
    UploadJSON(ctx *rrpc.Context, r io.Reader) error
    Upload2(ctx *rrpc.Context, info *UploadInfo, r io.Reader) error
    Upload2PDF(ctx *rrpc.Context, info *UploadInfo, r io.Reader) error
    Download(ctx *rrpc.Context, w io.Writer) error
    DownloadPDF(ctx *rrpc.Context, w io.Writer) error
}

// -- Errors ----------------------------------------------


var (
    ErrForbiddenError  = rrpc.RRPCError{Code: 2, Name: "ForbiddenError", Message: "forbidden", HTTPStatus: 403}

    ErrNotFoundError  = rrpc.RRPCError{Code: 1, Name: "NotFoundError", Message: "not found", HTTPStatus: 404}
)


