package webserver

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
)

const (
	// MaxFileSize limits uploaded file size to 100MB
	MaxFileSize = 100 * 1024 * 1024
	// MaxFormSize limits form data to 10MB
	MaxFormSize = 10 * 1024 * 1024
	// CSRFTokenLength defines the length of CSRF tokens
	CSRFTokenLength = 32
)

// AllowedFileExtensions defines the allowed file extensions for uploads
var AllowedFileExtensions = map[string]bool{
	".gcode": true,
	".gco":   true,
	".g":     true,
	".nc":    true,
	".txt":   true, // Allow .txt for testing
}

// ValidateFileUpload validates uploaded files for security
func ValidateFileUpload(file multipart.File, header *multipart.FileHeader) error {
	// Basic check for empty filename
	if strings.TrimSpace(header.Filename) == "" {
		return fmt.Errorf("filename cannot be empty")
	}

	// Check file size
	if header.Size > MaxFileSize {
		return fmt.Errorf("file too large: %d bytes (max %d)", header.Size, MaxFileSize)
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !AllowedFileExtensions[ext] {
		return fmt.Errorf("invalid file type: %s (allowed: %v)", ext, getAllowedExtensions())
	}

	// Check for path traversal in filename
	if strings.Contains(header.Filename, "..") || strings.Contains(header.Filename, "/") || strings.Contains(header.Filename, "\\") {
		return fmt.Errorf("invalid filename: contains path traversal characters")
	}

	// Read first few bytes to validate it's likely a text file (G-code)
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && n == 0 {
		return fmt.Errorf("cannot read file content")
	}

	// Reset file pointer
	if seeker, ok := file.(interface{ Seek(int64, int) (int64, error) }); ok {
		seeker.Seek(0, 0)
	}

	// Basic validation that it looks like G-code (contains printable ASCII)
	for i := 0; i < n; i++ {
		b := buffer[i]
		// Allow printable ASCII, newlines, carriage returns, and tabs
		if b < 32 && b != 10 && b != 13 && b != 9 {
			return fmt.Errorf("file contains invalid characters (not a text file)")
		}
	}

	return nil
}

// SanitizeString sanitizes user input to prevent XSS
func SanitizeString(input string) string {
	// HTML escape the input
	sanitized := html.EscapeString(input)
	// Trim whitespace
	sanitized = strings.TrimSpace(sanitized)
	return sanitized
}

// SanitizeFilename sanitizes filenames to prevent issues
func SanitizeFilename(filename string) string {
	// Remove any path separators and dangerous characters
	filename = strings.ReplaceAll(filename, "/", "")
	filename = strings.ReplaceAll(filename, "\\", "")
	filename = strings.ReplaceAll(filename, "..", "")
	filename = strings.ReplaceAll(filename, ":", "")
	filename = strings.ReplaceAll(filename, "*", "")
	filename = strings.ReplaceAll(filename, "?", "")
	filename = strings.ReplaceAll(filename, "<", "")
	filename = strings.ReplaceAll(filename, ">", "")
	filename = strings.ReplaceAll(filename, "|", "")
	filename = strings.TrimSpace(filename)
	
	// Ensure filename is not empty after sanitization
	if filename == "" {
		filename = "upload"
	}
	
	return filename
}

// GenerateCSRFToken generates a cryptographically secure CSRF token
func GenerateCSRFToken() (string, error) {
	bytes := make([]byte, CSRFTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate CSRF token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// ValidateCSRFToken validates a CSRF token from the request
func ValidateCSRFToken(r *http.Request, sessionToken string) bool {
	formToken := r.FormValue("csrf_token")
	return formToken != "" && formToken == sessionToken
}

// SetCSRFTokenCookie sets a CSRF token in a secure cookie
func SetCSRFTokenCookie(w http.ResponseWriter, token string) {
	cookie := &http.Cookie{
		Name:     "csrf_token",
		Value:    token,
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	}
	http.SetCookie(w, cookie)
}

// GetCSRFTokenFromCookie retrieves CSRF token from cookie
func GetCSRFTokenFromCookie(r *http.Request) string {
	cookie, err := r.Cookie("csrf_token")
	if err != nil {
		return ""
	}
	return cookie.Value
}

// getAllowedExtensions returns a slice of allowed extensions for error messages
func getAllowedExtensions() []string {
	exts := make([]string, 0, len(AllowedFileExtensions))
	for ext := range AllowedFileExtensions {
		exts = append(exts, ext)
	}
	return exts
}

// ValidateNumericInput validates numeric input within bounds
func ValidateNumericInput(value, min, max int64, fieldName string) error {
	if value < min {
		return fmt.Errorf("%s must be at least %d", fieldName, min)
	}
	if value > max {
		return fmt.Errorf("%s must be at most %d", fieldName, max)
	}
	return nil
}

// ValidateFloatInput validates float input within bounds
func ValidateFloatInput(value, min, max float64, fieldName string) error {
	if value < min {
		return fmt.Errorf("%s must be at least %.2f", fieldName, min)
	}
	if value > max {
		return fmt.Errorf("%s must be at most %.2f", fieldName, max)
	}
	return nil
}