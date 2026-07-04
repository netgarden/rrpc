package files

import (
    "github.com/netgarden/rrpc"

    "io"
)

// -- Structs ---------------------------------------------

type UploadDataInfo struct {
    Filename string `json:"filename"`
}

// -- Services --------------------------------------------

type FilesService interface {
    Upload(ctx *rrpc.Context, r io.Reader) error
    Upload2(ctx *rrpc.Context, info *UploadDataInfo, r io.Reader) error
    Download(ctx *rrpc.Context, w io.Writer) error
}

