package impl

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/netgarden/rrpc"
	"github.com/netgarden/rrpc-test/modules/files"
)

func NewFilesService(dir string) files.FilesService {
	if err := os.MkdirAll(dir, 0755); err != nil {
		panic(err)
	}
	return &FilesService{dir: dir}
}

type FilesService struct {
	dir string
}

func (s *FilesService) Upload(ctx *rrpc.Context, r io.Reader) error {
	filename := ctx.Params().GetString("filename")
	filePath := filepath.Join(s.dir, filepath.Base(filename))

	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, r)
	return err
}

func (s *FilesService) Upload2(ctx *rrpc.Context, info *files.UploadDataInfo, r io.Reader) error {
	filePath := filepath.Join(s.dir, filepath.Base(info.Filename))

	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, r)
	return err
}

func (s *FilesService) Download(ctx *rrpc.Context, w io.Writer) error {
	filename := ctx.Params().GetString("filename")
	filePath := filepath.Join(s.dir, filepath.Base(filename))

	f, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			ctx.Response().WriteHeader(http.StatusNotFound)
			return nil
		}
		return err
	}
	defer f.Close()

	_, err = io.Copy(w, f)
	return err
}