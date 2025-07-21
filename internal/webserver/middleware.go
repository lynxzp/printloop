package webserver

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/klauspost/compress/zstd"
)

type compressResponseWriter struct {
	http.ResponseWriter
	writer io.Writer
}

func (w *compressResponseWriter) Write(b []byte) (int, error) {
	return w.writer.Write(b)
}

func CompressionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check Accept-Encoding header
		acceptEncoding := r.Header.Get("Accept-Encoding")

		var writer io.Writer
		var encoding string

		if strings.Contains(acceptEncoding, "zstd") {
			w.Header().Set("Content-Encoding", "zstd")
			encoder, _ := zstd.NewWriter(w,
				zstd.WithEncoderLevel(zstd.SpeedBetterCompression),
				zstd.WithWindowSize(1<<23))
			defer encoder.Close()
			writer = encoder
			encoding = "zstd"
		} else if strings.Contains(acceptEncoding, "gzip") {
			w.Header().Set("Content-Encoding", "gzip")
			gz := gzip.NewWriter(w)
			defer gz.Close()
			writer = gz
			encoding = "gzip"
		}

		if encoding != "" {
			w.Header().Del("Content-Length") // Can't know compressed size
			cw := &compressResponseWriter{ResponseWriter: w, writer: writer}
			next.ServeHTTP(cw, r)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}
