package webserver

import (
	"compress/gzip"
	"io"
	"log/slog"
	"net/http"
	"net/url"
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

		var (
			writer   io.Writer
			encoding string
		)

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
			cw := &compressResponseWriter{ResponseWriter: w, writer: writer}
			next.ServeHTTP(cw, r)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

func LogPageRef(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		referer := r.Referer()
		if referer != "" {
			ref, _ := url.QueryUnescape(referer)

			refURL, err := url.Parse(ref)
			if err == nil {
				reqHost := r.Host

				if refURL.Host != "" && refURL.Host != reqHost {
					slog.Info("Hit the page", "url", r.URL.Path, "src", r.RemoteAddr, "ref", ref)
				} else {
					slog.Debug("Hit the page", "url", r.URL.Path, "src", r.RemoteAddr, "ref", ref)
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}
