package rrpc

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"time"
)

// ResponseInfo describes one finished request, as reported to a
// ResponseObserver. It covers every response the server produced, including
// error responses and requests that matched no route.
type ResponseInfo struct {
	// Route is the matched route's path template (service path + method
	// path, e.g. "/hooks/{token}"), never the raw request path, so it is
	// safe as a metrics label. It is "" when no route matched.
	Route string
	// Status is the HTTP status written (200 if the handler wrote none).
	// For a hijacked (WebSocket) connection it is 101.
	Status int
	// Bytes is the number of response body bytes written.
	Bytes int64
	// Duration is the time spent in ServeHTTP.
	Duration time.Duration
}

// ResponseObserver is called once per request, after the response has been
// written. It must not block and must not touch the request's body or the
// ResponseWriter.
type ResponseObserver func(r *http.Request, info ResponseInfo)

// Observe registers a ResponseObserver. Not safe to call concurrently with
// serving; register observers before the server starts.
func (s *Server) Observe(observers ...ResponseObserver) {
	s.observers = append(s.observers, observers...)
}

// recorder captures the status and size of a response. It forwards the
// optional interfaces the WebSocket upgrade needs.
type recorder struct {
	http.ResponseWriter
	status   int
	bytes    int64
	hijacked bool
}

func (r *recorder) WriteHeader(code int) {
	if r.status == 0 {
		r.status = code
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *recorder) Write(p []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(p)
	r.bytes += int64(n)
	return n, err
}

func (r *recorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (r *recorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("rrpc: the underlying ResponseWriter does not support hijacking")
	}
	conn, rw, err := h.Hijack()
	if err == nil {
		r.hijacked = true
	}
	return conn, rw, err
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (r *recorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

func (r *recorder) info(route string, d time.Duration) ResponseInfo {
	status := r.status
	switch {
	case r.hijacked:
		status = http.StatusSwitchingProtocols
	case status == 0:
		status = http.StatusOK
	}
	return ResponseInfo{Route: route, Status: status, Bytes: r.bytes, Duration: d}
}
