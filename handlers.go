package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"
)

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read HTML file
	htmlContent, err := os.ReadFile("www/index.html")
	if err != nil {
		slog.Error("Error reading index.html:", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(htmlContent)
}

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	log := slog.With("handler", "UploadHandler")
	log.Info("Received upload request", "remote_addr", r.RemoteAddr)

	req, err := receiveRequest(w, r)
	if err != nil {
		log.Error("Failed to receive request", "error", err)
		http.Error(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}
	inFileName := path.Join("uploads", req.FileName)
	outFileName := path.Join("results", req.FileName)

	defer os.Remove(inFileName)
	defer os.Remove(outFileName)

	err = ProcessFile(inFileName, outFileName, req)
	if err != nil {
		log.Error("Request processing failed", "error", err)
		http.Error(w, "File processing failed: "+err.Error(), http.StatusInternalServerError)
		log.Error("File processing failed", "error", err)
		return
	}

	if sendResponse(w, req) != nil {
		log.Error("Failed to send response", "error", err)
		http.Error(w, "Failed to send response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Info("Request processed", "filename", req.FileName)
}

func sendResponse(w http.ResponseWriter, req ProcessingRequest) error {
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"",
		path.Join("uploads", req.FileName)))
	w.Header().Set("Content-Type", "application/octet-stream")
	fileName := path.Join("results", req.FileName)
	resFile, err := os.Stat(fileName)
	if err != nil {
		return fmt.Errorf("failed to stat result file %s: %w", fileName, err)
	}
	w.Header().Set("Content-Length", strconv.FormatInt(resFile.Size(), 10))

	file, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("failed to open result file %s: %w", fileName, err)
	}

	writtenSize, err := io.Copy(w, file)
	if err != nil || writtenSize != resFile.Size() {
		return fmt.Errorf("failed writing response: %w", err)
	}
	return nil
}

func receiveRequest(w http.ResponseWriter, r *http.Request) (ProcessingRequest, error) {
	var req ProcessingRequest

	iterationsS := r.FormValue("iterations")
	var err error
	req.Iterations, err = strconv.ParseInt(iterationsS, 10, 64)
	if err != nil || req.Iterations <= 0 {
		return req, fmt.Errorf("invalid iterations value %v: %w", iterationsS, err)
	}
	waitTempS := r.FormValue("wait_temp")
	req.WaitTemp, err = strconv.ParseInt(waitTempS, 10, 64)
	if err != nil || req.WaitTemp < 0 {
		return req, fmt.Errorf("invalid wait_temp value %v: %w", waitTempS, err)
	}
	waitMinS := r.FormValue("wait_min")
	req.WaitMin, err = strconv.ParseInt(waitMinS, 10, 64)
	if err != nil || req.WaitMin < 0 {
		return req, fmt.Errorf("invalid wait_min value %v: %w", waitMinS, err)
	}

	const maxFileSize = 100 * 1024 * 1024
	r.Body = http.MaxBytesReader(w, r.Body, maxFileSize)

	err = r.ParseMultipartForm(maxFileSize)
	if err != nil {
		return req, fmt.Errorf("form parsing error: %w", err)
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		return req, fmt.Errorf("file retrieval error: %w", err)
	}
	defer file.Close()

	timestamp := time.Now().Unix()
	req.FileName = fmt.Sprintf("%d_%s", timestamp, header.Filename)
	filepath := path.Join("uploads", req.FileName)

	dst, err := os.Create(filepath)
	if err != nil {
		return req, fmt.Errorf("file creation failed: %w", err)
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		_ = os.Remove(filepath)
		return req, fmt.Errorf("file saving error: %w", err)
	}
	return req, nil
}
