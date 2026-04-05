package files

import (
    "github.com/netgarden/rrpc"

    "io"
)

// -- Services --------------------------------------------

type FilesService interface {
    Upload(ctx *rrpc.Context, r io.Reader) error
    Download(ctx *rrpc.Context, w io.Writer) error
}

