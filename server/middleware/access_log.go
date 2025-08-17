package middleware

import (
	"bufio"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// recorder counts status + bytes and preserves optional interfaces (SSE-safe).
type recorder struct {
	http.ResponseWriter
	status     int
	bytesWrite int
}

func newRecorder(w http.ResponseWriter) *recorder {
	return &recorder{ResponseWriter: w, status: http.StatusOK}
}

func (r *recorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *recorder) Write(p []byte) (int, error) {
	n, err := r.ResponseWriter.Write(p)
	r.bytesWrite += n
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
		return nil, nil, errors.New("hijacker not supported")
	}
	return h.Hijack()
}

func (r *recorder) Push(target string, opts *http.PushOptions) error {
	if p, ok := r.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}

// AccessLog logs request metadata for each request.
func AccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := newRecorder(w)
		next.ServeHTTP(rec, r)
		dur := time.Since(start)

		slog.Debug("access",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration", dur,
			"bytes", rec.bytesWrite,
		)
	})
}
