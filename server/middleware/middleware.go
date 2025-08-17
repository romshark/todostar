package middleware

import (
	"net/http"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/andybalholm/brotli/matchfinder"
)

// NoCache disables caching.
func NoCache(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		h.ServeHTTP(w, r)
	})
}

// Brotli uses brotli compression.
func Brotli(next http.Handler, compressionLevel int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "br") {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Encoding", "br")
		w.Header().Del("Content-Length")

		brWriter := brotli.NewWriterV2(w, compressionLevel)
		defer func() { _ = brWriter.Close() }()

		brw := &brotliResponseWriter{ResponseWriter: w, Writer: brWriter}
		next.ServeHTTP(brw, r)
	})
}

type brotliResponseWriter struct {
	http.ResponseWriter
	Writer *matchfinder.Writer
}

func (w *brotliResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}
