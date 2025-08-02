package main

import (
	"log/slog"
	"net/http"
	"os"
	"path"
	"printloop/internal/webserver"
	"strconv"
)

func main() {
	initLogger()

	// Initialize translations
	if err := webserver.LoadTranslations(); err != nil {
		slog.Error("Failed to load translations:", "err", err)
		return
	}

	if err := os.MkdirAll("files", 0755); err != nil {
		slog.Error("Failed to create files directory:", "err", err)
		return
	}

	if err := os.MkdirAll("files/uploads", 0755); err != nil {
		slog.Error("Failed to create files/uploads directory:", "err", err)
		return
	}

	if err := os.MkdirAll("files/results", 0755); err != nil {
		slog.Error("Failed to create files/results directory:", "err", err)
		return
	}

	mux := http.NewServeMux()

	// Setup routes
	mux.HandleFunc("/", webserver.HomeHandler)
	mux.HandleFunc("POST /upload", webserver.UploadHandler)
	mux.HandleFunc("/template", webserver.TemplateHandler)
	// Serve static files from embedded FS
	mux.Handle("/www/", http.StripPrefix("/www/", webserver.StaticFileServer()))

	handler := webserver.CompressionMiddleware(mux)

	slog.Info("Server started on port :8080")
	slog.Info("Open http://localhost:8080 in your browser")

	if err := http.ListenAndServe(":8080", handler); err != nil {
		slog.Error("Server startup error", "err", err)
		return
	}
}

func initLogger() {
	const useJSON = true
	var handler slog.Handler
	if !useJSON {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelInfo,
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.SourceKey {
					s := a.Value.Any().(*slog.Source)
					s.File = path.Base(s.File)
				}
				return a
			},
		})
	} else {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelInfo,
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.SourceKey {
					s := a.Value.Any().(*slog.Source)
					r := slog.String(slog.SourceKey, " "+path.Base(s.File)+":"+strconv.Itoa(s.Line)+" ")
					return r
				}
				return a
			},
		})
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}
