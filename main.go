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

	// Setup routes
	http.HandleFunc("/", webserver.HomeHandler)
	http.HandleFunc("POST /upload", webserver.UploadHandler)

	// Serve static files (CSS, JS, images)
	http.Handle("/www/", http.StripPrefix("/www/", http.FileServer(http.Dir("www/"))))

	slog.Info("Server started on port :8080")
	slog.Info("Open http://localhost:8080 in your browser")

	if err := http.ListenAndServe(":8080", nil); err != nil {
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
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
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
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
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
