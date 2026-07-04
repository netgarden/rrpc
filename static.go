package rrpc

import (
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type StaticFilesHandler struct {
	pathParamName string
	fs            fs.FS
	baseDir       string
}

func NewStaticFilesHandler(pathParamName string, fs fs.FS, baseDir string) *StaticFilesHandler {
	return &StaticFilesHandler{
		pathParamName: pathParamName,
		fs:            fs,
		baseDir:       baseDir,
	}
}

func (h *StaticFilesHandler) ServeStatic(ctx *Context, w io.Writer) error {

	path := ctx.Params().GetString(h.pathParamName)

	filePath := filepath.Join(h.baseDir, path)
	filePath = filepath.Clean(filePath)

	if !strings.HasPrefix(filePath, h.baseDir) {
		ctx.Response().WriteHeader(http.StatusNotFound)
		return nil
	}

	f, err := h.fs.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			ctx.Response().WriteHeader(http.StatusNotFound)
			return nil
		}
		return err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return err
	}
	fileSize := stat.Size()

	ext := filepath.Ext(filePath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	resp := ctx.Response()
	resp.Header().Set("Content-Type", contentType)
	resp.Header().Set("Accept-Ranges", "bytes")

	rangeHeader := ctx.Request().Header.Get("Range")
	if rangeHeader == "" {
		resp.Header().Set("Content-Length", strconv.FormatInt(fileSize, 10))
		_, err = io.Copy(w, f)
		return err
	}

	start, end, ok := parseByteRange(rangeHeader, fileSize)
	if !ok {
		resp.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", fileSize))
		resp.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
		return nil
	}

	seeker, canSeek := f.(io.Seeker)
	if !canSeek {
		resp.Header().Set("Content-Length", strconv.FormatInt(fileSize, 10))
		_, err = io.Copy(w, f)
		return err
	}

	if _, err := seeker.Seek(start, io.SeekStart); err != nil {
		return err
	}

	length := end - start + 1
	resp.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
	resp.Header().Set("Content-Length", strconv.FormatInt(length, 10))
	resp.WriteHeader(http.StatusPartialContent)

	_, err = io.Copy(w, io.LimitReader(f, length))
	return err
}

// parseByteRange parses a single Range header value (e.g. "bytes=0-1023")
// and returns the resolved start/end byte positions (inclusive).
// Multi-range requests are not supported and return ok=false.
func parseByteRange(rangeHeader string, fileSize int64) (start, end int64, ok bool) {
	if !strings.HasPrefix(rangeHeader, "bytes=") {
		return 0, 0, false
	}
	spec := strings.TrimPrefix(rangeHeader, "bytes=")
	if strings.Contains(spec, ",") {
		return 0, 0, false
	}
	parts := strings.SplitN(spec, "-", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	startStr, endStr := parts[0], parts[1]

	if startStr == "" {
		// suffix range: bytes=-N (last N bytes)
		suffix, err := strconv.ParseInt(endStr, 10, 64)
		if err != nil || suffix <= 0 {
			return 0, 0, false
		}
		start = fileSize - suffix
		if start < 0 {
			start = 0
		}
		end = fileSize - 1
	} else {
		s, err := strconv.ParseInt(startStr, 10, 64)
		if err != nil || s < 0 {
			return 0, 0, false
		}
		start = s
		if endStr == "" {
			end = fileSize - 1
		} else {
			e, err := strconv.ParseInt(endStr, 10, 64)
			if err != nil || e < start {
				return 0, 0, false
			}
			end = e
			if end >= fileSize {
				end = fileSize - 1
			}
		}
	}

	if start >= fileSize {
		return 0, 0, false
	}

	return start, end, true
}