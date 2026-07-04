package rrpc

import (
	"io"
	"net/http"
)

func NewResponseWriter(w http.ResponseWriter, contentType string) io.Writer {
	return &responseWriter{w: w, contentType: contentType}
}

type responseWriter struct {
	w           http.ResponseWriter
	contentType string
	written     bool
}

func (r *responseWriter) Write(p []byte) (n int, err error) {
	if !r.written {
		if r.w.Header().Get("Content-Type") == "" {
			r.w.Header().Set("Content-Type", r.contentType)
		}
		r.written = true
	}
	return r.w.Write(p)
}