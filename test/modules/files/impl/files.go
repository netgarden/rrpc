package impl

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/netgarden/rrpc"
	"github.com/netgarden/rrpc/test/modules/files"
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

// UploadJSON is identical to Upload; it exists to exercise
// @RequestContentType("application/json") on a single-bytes method
// end-to-end (see rrpc/server_test.go's TestServer_* Content-Type tests).
func (s *FilesService) UploadJSON(ctx *rrpc.Context, r io.Reader) error {
	return s.Upload(ctx, r)
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

// Upload2PDF is identical to Upload2; it exists to exercise
// @RequestContentType("application/pdf") on a multipart method's bytes part
// end-to-end.
func (s *FilesService) Upload2PDF(ctx *rrpc.Context, info *files.UploadDataInfo, r io.Reader) error {
	return s.Upload2(ctx, info, r)
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

// DownloadPDF is identical to Download; it exists to exercise
// @ResponseContentType("application/pdf") end-to-end.
func (s *FilesService) DownloadPDF(ctx *rrpc.Context, w io.Writer) error {
	return s.Download(ctx, w)
}