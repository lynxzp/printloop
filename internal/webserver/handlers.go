package webserver

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path"
	"printloop/internal/processor"
	"strconv"
	"strings"
	"time"
)

//go:embed www/*
var wwwFiles embed.FS

// TemplateData holds data for template rendering
type TemplateData struct {
	Lang string
	T    Translation
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Determine language
	lang := GetLanguageFromRequest(r)

	// Get translations for the determined language
	translations := GetTranslations(lang)

	// Create template data
	data := TemplateData{
		Lang: lang,
		T:    translations,
	}

	// Read template file
	templateContent, err := wwwFiles.ReadFile("www/index_template.html")
	if err != nil {
		slog.Error("Error reading index_template.html:", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)

		return
	}

	// Parse and execute template
	tmpl, err := template.New("index").Parse(string(templateContent))
	if err != nil {
		slog.Error("Error parsing template:", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	err = tmpl.Execute(w, data)
	if err != nil {
		slog.Error("Error executing template:", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)

		return
	}
}

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	log := slog.With("handler", "UploadHandler")
	log.Info("Received upload request", "remote_addr", r.RemoteAddr)

	// Determine language for error messages
	lang := GetLanguageFromRequest(r)

	req, err := receiveRequest(w, r)
	if err != nil {
		log.Error("Failed to receive request", "error", err)
		WriteErrorResponseWithLang(w, err, http.StatusBadRequest, lang)

		return
	}

	inFileName := path.Join("files/uploads", req.FileName)
	outFileName := path.Join("files/results", req.FileName)

	defer os.Remove(inFileName)
	defer os.Remove(outFileName)

	err = processor.ProcessFile(inFileName, outFileName, req)
	if err != nil {
		log.Error("Request processing failed", "error", err)
		WriteErrorResponseWithLang(w, err, http.StatusInternalServerError, lang)

		return
	}

	err = sendResponse(w, req)
	if err != nil {
		log.Error("Failed to send response", "error", err)
		WriteErrorResponseWithLang(w, err, http.StatusInternalServerError, lang)

		return
	}

	log.Info("Request processed", "filename", req.FileName)
}

func sendResponse(w http.ResponseWriter, req processor.ProcessingRequest) error {
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", req.FileName))
	w.Header().Set("Content-Type", "application/octet-stream")

	fileName := path.Join("files/results", req.FileName)

	file, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("failed to open result file %s: %w", fileName, err)
	}
	defer file.Close()

	_, err = io.Copy(w, file)
	if err != nil {
		return fmt.Errorf("failed writing response: %w", err)
	}

	return nil
}

func receiveRequest(w http.ResponseWriter, r *http.Request) (processor.ProcessingRequest, error) {
	var req processor.ProcessingRequest

	const maxFileSize = 1024 * 1024 * 1024

	r.Body = http.MaxBytesReader(w, r.Body, maxFileSize)

	err := r.ParseMultipartForm(1024 * 1024) // receive up to 1MB of form data
	if err != nil {
		return req, fmt.Errorf("form parsing error: %w", err)
	}

	iterationsS := r.FormValue("iterations")

	req.Iterations, err = strconv.ParseInt(iterationsS, 10, 64)

	if err != nil || req.Iterations < 2 || req.Iterations > 10000 {
		return req, fmt.Errorf("invalid iterations value %v: must be between 2 and 10000", iterationsS)
	}

	waitBedCooldownTempS := r.FormValue("waitBedCooldownTemp")

	req.WaitBedCooldownTemp, err = strconv.ParseInt(waitBedCooldownTempS, 10, 64)
	if (err != nil || req.WaitBedCooldownTemp < 0) && waitBedCooldownTempS != "" {
		return req, fmt.Errorf("invalid wait_temp value %v: %w", waitBedCooldownTempS, err)
	}

	waitMinS := r.FormValue("wait_min")

	req.WaitMin, err = strconv.ParseInt(waitMinS, 10, 64)
	if (err != nil || req.WaitMin < 0) && waitMinS != "" {
		return req, fmt.Errorf("invalid wait_min value %v: %w", waitMinS, err)
	}

	extraExtrudeS := r.FormValue("extra_extrude")

	req.ExtraExtrude, err = strconv.ParseFloat(extraExtrudeS, 64)
	if (err != nil || req.ExtraExtrude < 0) && extraExtrudeS != "" {
		return req, fmt.Errorf("invalid extra_extrude value %v: %w", waitMinS, err)
	}

	req.Printer = r.FormValue("printer")

	// Handle custom template if provided
	customTemplate := r.FormValue("custom_template")
	if customTemplate != "" {
		req.CustomTemplate = strings.TrimSpace(customTemplate)
	}

	// Handle test print with pause option
	req.TestPrintWithPause = r.FormValue("test_print_pause") == "true"

	file, header, err := r.FormFile("file")
	if err != nil {
		return req, fmt.Errorf("file retrieval error: %w", err)
	}
	defer file.Close()

	timestamp := time.Now().Unix()
	req.FileName = fmt.Sprintf("%d_%s", timestamp, header.Filename)
	filepath := path.Join("files/uploads", req.FileName)

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

func TemplateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	printerName := r.URL.Query().Get("printer")
	if printerName == "" {
		http.Error(w, "Missing printer parameter", http.StatusBadRequest)
		return
	}

	// Normalize printer name (same logic as in processor)
	printerName = strings.ReplaceAll(printerName, " ", "-")
	printerName = strings.ToLower(printerName)

	// Load printer definition
	printerDef, err := processor.LoadPrinterDefinitionPublic(printerName)
	if err != nil {
		http.Error(w, "Printer not found: "+err.Error(), http.StatusNotFound)
		return
	}

	// Convert the complete printer definition to TOML format with custom handling for multiline strings
	var buf strings.Builder

	// Write basic fields
	fmt.Fprintf(&buf, "Name = %q\n\n", printerDef.Name)

	// Write Markers section
	buf.WriteString("[Markers]\n")
	fmt.Fprintf(&buf, "EndInitSection = %v\n", formatStringSlice(printerDef.Markers.EndInitSection))
	fmt.Fprintf(&buf, "EndPrintSection = %v\n", formatStringSlice(printerDef.Markers.EndPrintSection))
	buf.WriteString("\n")

	// Write SearchStrategy section
	buf.WriteString("[SearchStrategy]\n")
	fmt.Fprintf(&buf, "EndInitSectionStrategy = %q\n", printerDef.SearchStrategy.EndInitSectionStrategy)
	fmt.Fprintf(&buf, "EndPrintSectionStrategy = %q\n", printerDef.SearchStrategy.EndPrintSectionStrategy)
	buf.WriteString("\n")

	// Write Parameters section
	if len(printerDef.Parameters) > 0 {
		buf.WriteString("[Parameters]\n")

		for key, value := range printerDef.Parameters {
			fmt.Fprintf(&buf, "%s = %v\n", key, formatValue(value))
		}

		buf.WriteString("\n")
	}

	// Write Template section with multiline string
	buf.WriteString("[Template]\n")
	buf.WriteString("Code = \"\"\"\n")
	buf.WriteString(printerDef.Template.Code)

	if !strings.HasSuffix(printerDef.Template.Code, "\n") {
		buf.WriteString("\n")
	}

	buf.WriteString("\"\"\"\n")

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(buf.String()))
}

func formatStringSlice(slice []string) string {
	if len(slice) == 0 {
		return "[]"
	}

	if len(slice) == 1 {
		return fmt.Sprintf("[%q]", slice[0])
	}

	parts := make([]string, 0, len(slice))
	for _, s := range slice {
		parts = append(parts, fmt.Sprintf("%q", s))
	}

	return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
}

func formatValue(v any) string {
	switch val := v.(type) {
	case string:
		return fmt.Sprintf("%q", val)
	case int, int64, float64:
		return fmt.Sprintf("%v", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func StaticFileServer() http.Handler {
	subFS, err := fs.Sub(wwwFiles, "www")
	if err != nil {
		slog.Error("Failed to create sub-filesystem", "error", err)
		return http.FileServer(http.FS(wwwFiles))
	}

	return http.FileServer(http.FS(subFS))
}

func FaviconHandler(filePath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := wwwFiles.ReadFile(filePath)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		// Set appropriate content type
		if strings.HasSuffix(filePath, ".ico") {
			w.Header().Set("Content-Type", "image/x-icon")
		} else if strings.HasSuffix(filePath, ".png") {
			w.Header().Set("Content-Type", "image/png")
		}

		// Set cache headers for favicons
		w.Header().Set("Cache-Control", "public, max-age=31536000") // 1 year

		_, _ = w.Write(data)
	}
}

// HintHandler serves hint text for the UI tooltips
func HintHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get hint key and language from query parameters
	hintKey := r.URL.Query().Get("key")
	if hintKey == "" {
		http.Error(w, "Hint key is required", http.StatusBadRequest)
		return
	}

	// Determine language
	lang := GetLanguageFromRequest(r)

	// Get the hint text
	hintText := GetTranslation(lang, hintKey)
	if hintText == hintKey {
		// If translation not found, return a default message
		if lang == "uk" {
			hintText = "Інформація недоступна"
		} else {
			hintText = "Information not available"
		}
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(hintText))
}
