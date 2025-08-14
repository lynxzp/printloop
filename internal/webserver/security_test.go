package webserver

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecurity(t *testing.T) {
	t.Run("ValidateFileUpload", func(t *testing.T) {
		tests := []struct {
			name        string
			filename    string
			content     string
			expectError bool
			errorMatch  string
		}{
			{
				name:        "valid gcode file",
				filename:    "test.gcode",
				content:     "G1 X10 Y10\nG1 Z5\n",
				expectError: false,
			},
			{
				name:        "invalid extension",
				filename:    "test.exe",
				content:     "content",
				expectError: true,
				errorMatch:  "invalid file type",
			},
			{
				name:        "path traversal filename",
				filename:    "test..gcode",
				content:     "content",
				expectError: true,
				errorMatch:  "path traversal",
			},
			{
				name:        "whitespace only filename",
				filename:    "   ",
				content:     "content",
				expectError: true,
				errorMatch:  "cannot be empty",
			},
			{
				name:        "binary content",
				filename:    "test.gcode",
				content:     "\x00\x01\x02\x03",
				expectError: true,
				errorMatch:  "invalid characters",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Create a fake multipart file
				var b bytes.Buffer
				writer := multipart.NewWriter(&b)
				part, _ := writer.CreateFormFile("file", tt.filename)
				part.Write([]byte(tt.content))
				writer.Close()

				// Extract the file from the form
				req := httptest.NewRequest("POST", "/", &b)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				req.ParseMultipartForm(1024 * 1024)
				
				file, header, err := req.FormFile("file")
				if err != nil {
					t.Fatalf("Failed to get form file: %v", err)
				}
				defer file.Close()

				err = ValidateFileUpload(file, header)
				if tt.expectError {
					assert.Error(t, err)
					if tt.errorMatch != "" {
						assert.Contains(t, err.Error(), tt.errorMatch)
					}
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("SanitizeString", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"normal text", "normal text"},
			{"<script>alert('xss')</script>", "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"},
			{"  whitespace  ", "whitespace"},
			{"<b>bold</b>", "&lt;b&gt;bold&lt;/b&gt;"},
		}

		for _, tt := range tests {
			result := SanitizeString(tt.input)
			assert.Equal(t, tt.expected, result)
		}
	})

	t.Run("SanitizeFilename", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"normal.gcode", "normal.gcode"},
			{"../../../etc/passwd", "etcpasswd"},
			{"file/with\\slashes", "filewithslashes"},
			{"file:with*dangerous?chars", "filewithdangerouschars"},
			{"", "upload"},
		}

		for _, tt := range tests {
			result := SanitizeFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		}
	})

	t.Run("CSRF Token Generation and Validation", func(t *testing.T) {
		// Test token generation
		token1, err := GenerateCSRFToken()
		assert.NoError(t, err)
		assert.NotEmpty(t, token1)
		assert.Len(t, token1, CSRFTokenLength*2) // hex encoded

		token2, err := GenerateCSRFToken()
		assert.NoError(t, err)
		assert.NotEqual(t, token1, token2) // Should be unique

		// Test token validation
		req := httptest.NewRequest("POST", "/", strings.NewReader("csrf_token="+token1))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.ParseForm()

		assert.True(t, ValidateCSRFToken(req, token1))
		assert.False(t, ValidateCSRFToken(req, token2))
		assert.False(t, ValidateCSRFToken(req, "invalid"))
	})

	t.Run("Numeric Input Validation", func(t *testing.T) {
		tests := []struct {
			name        string
			value       int64
			min         int64
			max         int64
			field       string
			expectError bool
		}{
			{"valid value", 50, 1, 100, "test", false},
			{"minimum value", 1, 1, 100, "test", false},
			{"maximum value", 100, 1, 100, "test", false},
			{"below minimum", 0, 1, 100, "test", true},
			{"above maximum", 101, 1, 100, "test", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := ValidateNumericInput(tt.value, tt.min, tt.max, tt.field)
				if tt.expectError {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tt.field)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("Float Input Validation", func(t *testing.T) {
		tests := []struct {
			name        string
			value       float64
			min         float64
			max         float64
			field       string
			expectError bool
		}{
			{"valid value", 2.5, 0.0, 5.0, "test", false},
			{"minimum value", 0.0, 0.0, 5.0, "test", false},
			{"maximum value", 5.0, 0.0, 5.0, "test", false},
			{"below minimum", -0.1, 0.0, 5.0, "test", true},
			{"above maximum", 5.1, 0.0, 5.0, "test", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := ValidateFloatInput(tt.value, tt.min, tt.max, tt.field)
				if tt.expectError {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tt.field)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})
}