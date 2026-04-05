package test

import (
    "github.com/netgarden/rrpc"

    "io"
)

// -- Services --------------------------------------------

type PingerService interface {
    Ping(ctx *rrpc.Context) error
}

type StaticService interface {
    ServeStatic(ctx *rrpc.Context, w io.Writer) error
}

