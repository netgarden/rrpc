package impl

import (
	"io"
	"os"

	"github.com/netgarden/rrpc"
	"github.com/netgarden/rrpc/test/modules/test"
)

func NewStaticService(dir string) test.StaticService {
	handler := rrpc.NewStaticFilesHandler("filepath", os.DirFS(dir), "/")
	return &StaticService{
		handler: handler,
	}
}

type StaticService struct {
	handler *rrpc.StaticFilesHandler
}

func (s *StaticService) ServeStatic(ctx *rrpc.Context, w io.Writer) error {
	return s.handler.ServeStatic(ctx, w)
}
