package impl

import (
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/netgarden/rrpc"
	"github.com/netgarden/rrpc-test/modules/test"
)

func NewStaticService(dir string) test.StaticService {
	return &StaticService{dir: dir}
}

type StaticService struct {
	dir string
}

func (s *StaticService) ServeStatic(ctx *rrpc.Context, w io.Writer) error {

	path := ctx.Params().GetString("filepath")

	filePath := filepath.Join(s.dir, filepath.Clean("/"+path))

	f, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			ctx.Response().WriteHeader(http.StatusNotFound)
			return nil
		}
		return err
	}
	defer f.Close()

	ext := filepath.Ext(filePath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	ctx.Response().Header().Set("Content-Type", contentType)

	_, err = io.Copy(w, f)
	return err
}
