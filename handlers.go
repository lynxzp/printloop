package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// getContentType returns the appropriate MIME type for the file format
func getContentType(format string) string {
	switch format {
	case "json":
		return "application/json"
	case "csv":
		return "text/csv"
	case "txt":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}

// HomeHandler serves the main page with form
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read HTML file
	htmlContent, err := os.ReadFile("www/index.html")
	if err != nil {
		log.Printf("Error reading HTML file: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(htmlContent)
}

// UploadHandler handles file upload and processing
func UploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form (max 10MB)
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Form parsing error: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get file from form
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "File retrieval error: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Get form parameters
	operation := r.FormValue("operation")
	format := r.FormValue("format")
	options := r.FormValue("options")

	if operation == "" || format == "" {
		http.Error(w, "Operation and format are required", http.StatusBadRequest)
		return
	}

	// Create unique filename
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("%d_%s", timestamp, header.Filename)
	filepath := filepath.Join("uploads", filename)

	// Save uploaded file
	dst, err := os.Create(filepath)
	if err != nil {
		http.Error(w, "File creation error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "File saving error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create processing request
	request := ProcessingRequest{
		InputFile: filepath,
		Operation: operation,
		Format:    format,
		Options:   options,
		Timestamp: timestamp,
	}

	// Process file
	result, err := ProcessFile(request)
	if err != nil {
		log.Printf("File processing error: %v", err)
		http.Error(w, "File processing error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set headers for immediate download
	downloadFilename := fmt.Sprintf("processed_%s_%s.%s",
		operation,
		strings.TrimSuffix(header.Filename, path.Ext(header.Filename)),
		format)

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", downloadFilename))
	w.Header().Set("Content-Type", getContentType(format))
	w.Header().Set("Content-Length", strconv.Itoa(len(result.Data)))

	// Write processed data directly to response
	_, err = w.Write(result.Data)
	if err != nil {
		log.Printf("Error writing response: %v", err)
		return
	}

	log.Printf("File processed and downloaded: %s -> %s", header.Filename, downloadFilename)
}

// DownloadHandler serves the result file for download
func DownloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get filename from URL
	filename := strings.TrimPrefix(r.URL.Path, "/download/")
	if filename == "" {
		http.Error(w, "Filename not specified", http.StatusBadRequest)
		return
	}

	filepath := filepath.Join("results", filename)

	// Check if file exists
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Open file
	file, err := os.Open(filepath)
	if err != nil {
		http.Error(w, "File opening error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Get file info
	stat, err := file.Stat()
	if err != nil {
		http.Error(w, "File info error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set headers for download
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))

	// Copy file to response
	_, err = io.Copy(w, file)
	if err != nil {
		log.Printf("File transfer error: %v", err)
	}
}
