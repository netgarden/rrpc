package echo

import (
    "github.com/netgarden/rrpc"
)

// -- Structs ---------------------------------------------

type EchoRequest struct {
    Message string `json:"message"`
    Count int `json:"count"`
    Flag bool `json:"flag"`
}

type EchoResponse struct {
    Result string `json:"result"`
}

// -- Services --------------------------------------------

type EchoService interface {
    Echo(ctx *rrpc.Context, request *EchoRequest) (*EchoResponse, error)
}

