// file: internal/webserver/handlers_test.go
package webserver

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"printloop/internal/processor"
	"strings"
	"testing"
)

func TestHomeHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
		checkResponse  func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name:           "valid GET request",
			method:         "GET",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
				// Check that we got some HTML content (the embedded file)
				body := w.Body.String()
				assert.Contains(t, body, "<html")
				assert.Contains(t, body, "Continuous loop 3D printing")
			},
		},
		{
			name:           "invalid method",
			method:         "POST",
			expectedStatus: http.StatusMethodNotAllowed,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, "Method not allowed\n", w.Body.String())
			},
		},
		{
			name:           "PUT method not allowed",
			method:         "PUT",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "DELETE method not allowed",
			method:         "DELETE",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/", nil)
			w := httptest.NewRecorder()

			HomeHandler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestUploadHandler(t *testing.T) {
	// Setup test directories
	setupTestDirs := func(t *testing.T) {
		err := os.MkdirAll("files/uploads", 0755)
		require.NoError(t, err)
		err = os.MkdirAll("files/results", 0755)
		require.NoError(t, err)
		t.Cleanup(func() {
			os.RemoveAll("files")
		})
	}

	tests := []struct {
		name           string
		setupRequest   func(t *testing.T) *http.Request
		expectedStatus int
		checkResponse  func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "invalid form data",
			setupRequest: func(t *testing.T) *http.Request {
				req := httptest.NewRequest("POST", "/upload", strings.NewReader("invalid"))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return req
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "Invalid request")
			},
		},
		{
			name: "missing file",
			setupRequest: func(t *testing.T) *http.Request {
				var buf bytes.Buffer
				writer := multipart.NewWriter(&buf)
				writer.WriteField("iterations", "5")
				writer.Close()

				req := httptest.NewRequest("POST", "/upload", &buf)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				return req
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "Invalid request")
			},
		},
		{
			name: "invalid iterations",
			setupRequest: func(t *testing.T) *http.Request {
				return createUploadRequestWithParams(t, map[string]string{
					"iterations": "invalid",
				})
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "Invalid request")
			},
		},
		{
			name: "large file within limit",
			setupRequest: func(t *testing.T) *http.Request {
				var buf bytes.Buffer
				writer := multipart.NewWriter(&buf)
				writer.WriteField("iterations", "1")

				part, err := writer.CreateFormFile("file", "large.txt")
				require.NoError(t, err)
				// Write a moderately large file (1KB)
				largeContent := strings.Repeat("test data line\n", 64)
				part.Write([]byte(largeContent))
				writer.Close()

				req := httptest.NewRequest("POST", "/upload", &buf)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				return req
			},
			expectedStatus: http.StatusInternalServerError, // Will fail in processor
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "File processing failed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestDirs(t)

			req := tt.setupRequest(t)
			w := httptest.NewRecorder()

			UploadHandler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestSendResponse(t *testing.T) {
	tests := []struct {
		name           string
		setupFile      func(t *testing.T) processor.ProcessingRequest
		expectedStatus int
		checkResponse  func(t *testing.T, w *httptest.ResponseRecorder, req processor.ProcessingRequest)
	}{
		{
			name: "valid file send",
			setupFile: func(t *testing.T) processor.ProcessingRequest {
				err := os.MkdirAll("files/results", 0755)
				require.NoError(t, err)
				t.Cleanup(func() { os.RemoveAll("files") })

				fileName := "test_file.txt"
				content := "test content"
				filePath := path.Join("files/results", fileName)
				err = os.WriteFile(filePath, []byte(content), 0644)
				require.NoError(t, err)

				return processor.ProcessingRequest{FileName: fileName}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder, req processor.ProcessingRequest) {
				assert.Equal(t, "application/octet-stream", w.Header().Get("Content-Type"))
				assert.Contains(t, w.Header().Get("Content-Disposition"), req.FileName)
				assert.Equal(t, "12", w.Header().Get("Content-Length")) // "test content" is 12 bytes
				assert.Equal(t, "test content", w.Body.String())
			},
		},
		{
			name: "file not found",
			setupFile: func(t *testing.T) processor.ProcessingRequest {
				err := os.MkdirAll("files/results", 0755)
				require.NoError(t, err)
				t.Cleanup(func() { os.RemoveAll("files") })

				return processor.ProcessingRequest{FileName: "nonexistent.txt"}
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "empty file",
			setupFile: func(t *testing.T) processor.ProcessingRequest {
				err := os.MkdirAll("files/results", 0755)
				require.NoError(t, err)
				t.Cleanup(func() { os.RemoveAll("files") })

				fileName := "empty_file.txt"
				filePath := path.Join("files/results", fileName)
				err = os.WriteFile(filePath, []byte(""), 0644)
				require.NoError(t, err)

				return processor.ProcessingRequest{FileName: fileName}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder, req processor.ProcessingRequest) {
				assert.Equal(t, "0", w.Header().Get("Content-Length"))
				assert.Equal(t, "", w.Body.String())
			},
		},
		{
			name: "special characters in filename",
			setupFile: func(t *testing.T) processor.ProcessingRequest {
				err := os.MkdirAll("files/results", 0755)
				require.NoError(t, err)
				t.Cleanup(func() { os.RemoveAll("files") })

				fileName := "test file with spaces & symbols.txt"
				content := "special content"
				filePath := path.Join("files/results", fileName)
				err = os.WriteFile(filePath, []byte(content), 0644)
				require.NoError(t, err)

				return processor.ProcessingRequest{FileName: fileName}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder, req processor.ProcessingRequest) {
				assert.Contains(t, w.Header().Get("Content-Disposition"), req.FileName)
				assert.Equal(t, "special content", w.Body.String())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupFile(t)
			w := httptest.NewRecorder()

			err := sendResponse(w, req)

			if tt.expectedStatus == http.StatusOK {
				assert.NoError(t, err)
				if tt.checkResponse != nil {
					tt.checkResponse(t, w, req)
				}
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestReceiveRequest(t *testing.T) {
	setupTestDirs := func(t *testing.T) {
		err := os.MkdirAll("files/uploads", 0755)
		require.NoError(t, err)
		t.Cleanup(func() {
			os.RemoveAll("files")
		})
	}

	tests := []struct {
		name          string
		setupRequest  func(t *testing.T) *http.Request
		expectedError bool
		validateReq   func(t *testing.T, req processor.ProcessingRequest)
	}{
		{
			name: "valid request",
			setupRequest: func(t *testing.T) *http.Request {
				return createValidUploadRequest(t)
			},
			expectedError: false,
			validateReq: func(t *testing.T, req processor.ProcessingRequest) {
				assert.Equal(t, int64(5), req.Iterations)
				assert.Equal(t, int64(200), req.WaitTemp)
				assert.Equal(t, int64(60), req.WaitMin)
				assert.Equal(t, 0.1, req.ExtraExtrude)
				assert.Equal(t, "test_printer", req.Printer)
				assert.Contains(t, req.FileName, "test.txt")
			},
		},
		{
			name: "invalid iterations",
			setupRequest: func(t *testing.T) *http.Request {
				return createUploadRequestWithParams(t, map[string]string{
					"iterations": "invalid",
				})
			},
			expectedError: true,
		},
		{
			name: "zero iterations",
			setupRequest: func(t *testing.T) *http.Request {
				return createUploadRequestWithParams(t, map[string]string{
					"iterations": "0",
				})
			},
			expectedError: true,
		},
		{
			name: "invalid wait_temp",
			setupRequest: func(t *testing.T) *http.Request {
				return createUploadRequestWithParams(t, map[string]string{
					"iterations": "5",
					"wait_temp":  "invalid",
				})
			},
			expectedError: true,
		},
		{
			name: "negative wait_temp",
			setupRequest: func(t *testing.T) *http.Request {
				return createUploadRequestWithParams(t, map[string]string{
					"iterations": "5",
					"wait_temp":  "-1",
				})
			},
			expectedError: true,
		},
		{
			name: "invalid wait_min",
			setupRequest: func(t *testing.T) *http.Request {
				return createUploadRequestWithParams(t, map[string]string{
					"iterations": "5",
					"wait_min":   "invalid",
				})
			},
			expectedError: true,
		},
		{
			name: "negative wait_min",
			setupRequest: func(t *testing.T) *http.Request {
				return createUploadRequestWithParams(t, map[string]string{
					"iterations": "5",
					"wait_min":   "-1",
				})
			},
			expectedError: true,
		},
		{
			name: "invalid extra_extrude",
			setupRequest: func(t *testing.T) *http.Request {
				return createUploadRequestWithParams(t, map[string]string{
					"iterations":    "5",
					"extra_extrude": "invalid",
				})
			},
			expectedError: true,
		},
		{
			name: "negative extra_extrude",
			setupRequest: func(t *testing.T) *http.Request {
				return createUploadRequestWithParams(t, map[string]string{
					"iterations":    "5",
					"extra_extrude": "-1",
				})
			},
			expectedError: true,
		},
		{
			name: "missing file",
			setupRequest: func(t *testing.T) *http.Request {
				var buf bytes.Buffer
				writer := multipart.NewWriter(&buf)

				writer.WriteField("iterations", "5")
				writer.Close()

				req := httptest.NewRequest("POST", "/upload", &buf)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				return req
			},
			expectedError: true,
		},
		{
			name: "empty optional fields",
			setupRequest: func(t *testing.T) *http.Request {
				return createUploadRequestWithParams(t, map[string]string{
					"iterations":    "5",
					"wait_temp":     "",
					"wait_min":      "",
					"extra_extrude": "",
					"printer":       "",
				})
			},
			expectedError: false,
			validateReq: func(t *testing.T, req processor.ProcessingRequest) {
				assert.Equal(t, int64(5), req.Iterations)
				assert.Equal(t, int64(0), req.WaitTemp)
				assert.Equal(t, int64(0), req.WaitMin)
				assert.Equal(t, 0.0, req.ExtraExtrude)
				assert.Equal(t, "", req.Printer)
			},
		},
		{
			name: "custom template with whitespace",
			setupRequest: func(t *testing.T) *http.Request {
				return createUploadRequestWithParams(t, map[string]string{
					"iterations":      "5",
					"custom_template": "  \n  G1 X10 Y10  \n  G1 Z5  \n  ",
				})
			},
			expectedError: false,
			validateReq: func(t *testing.T, req processor.ProcessingRequest) {
				assert.Equal(t, "G1 X10 Y10  \n  G1 Z5", req.CustomTemplate)
			},
		},
		{
			name: "very large iterations",
			setupRequest: func(t *testing.T) *http.Request {
				return createUploadRequestWithParams(t, map[string]string{
					"iterations": "9223372036854775807", // max int64
				})
			},
			expectedError: false,
			validateReq: func(t *testing.T, req processor.ProcessingRequest) {
				assert.Equal(t, int64(9223372036854775807), req.Iterations)
			},
		},
		{
			name: "filename with special characters",
			setupRequest: func(t *testing.T) *http.Request {
				var buf bytes.Buffer
				writer := multipart.NewWriter(&buf)
				writer.WriteField("iterations", "5")

				part, err := writer.CreateFormFile("file", "test file with spaces & symbols.gcode")
				require.NoError(t, err)
				part.Write([]byte("test content"))
				writer.Close()

				req := httptest.NewRequest("POST", "/upload", &buf)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				return req
			},
			expectedError: false,
			validateReq: func(t *testing.T, req processor.ProcessingRequest) {
				assert.Contains(t, req.FileName, "test file with spaces & symbols.gcode")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestDirs(t)

			req := tt.setupRequest(t)
			w := httptest.NewRecorder()

			result, err := receiveRequest(w, req)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validateReq != nil {
					tt.validateReq(t, result)
				}
			}
		})
	}
}

func TestTemplateHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		queryParams    string
		expectedStatus int
		checkResponse  func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name:           "invalid method POST",
			method:         "POST",
			queryParams:    "?printer=test",
			expectedStatus: http.StatusMethodNotAllowed,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, "Method not allowed\n", w.Body.String())
			},
		},
		{
			name:           "invalid method PUT",
			method:         "PUT",
			queryParams:    "?printer=test",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "missing printer parameter",
			method:         "GET",
			queryParams:    "",
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, "Missing printer parameter\n", w.Body.String())
			},
		},
		{
			name:           "empty printer parameter",
			method:         "GET",
			queryParams:    "?printer=",
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, "Missing printer parameter\n", w.Body.String())
			},
		},
		{
			name:           "nonexistent printer",
			method:         "GET",
			queryParams:    "?printer=nonexistent_printer",
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "Printer not found")
			},
		},
		{
			name:           "printer name with spaces",
			method:         "GET",
			queryParams:    "?printer=Test%20Printer%20Name",
			expectedStatus: http.StatusNotFound, // Will normalize to test-printer-name and likely not exist
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "Printer not found")
			},
		},
		{
			name:           "printer name case insensitive",
			method:         "GET",
			queryParams:    "?printer=TEST_PRINTER",
			expectedStatus: http.StatusNotFound, // Will normalize to test_printer and likely not exist
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "Printer not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/template"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			TemplateHandler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestFormatStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{
			name:     "empty slice",
			input:    []string{},
			expected: "[]",
		},
		{
			name:     "single element",
			input:    []string{"test"},
			expected: `["test"]`,
		},
		{
			name:     "multiple elements",
			input:    []string{"first", "second", "third"},
			expected: `["first", "second", "third"]`,
		},
		{
			name:     "element with quotes",
			input:    []string{`test"quote`},
			expected: `["test\"quote"]`,
		},
		{
			name:     "element with special characters",
			input:    []string{"test\nwith\ttabs"},
			expected: `["test\nwith\ttabs"]`,
		},
		{
			name:     "empty string element",
			input:    []string{""},
			expected: `[""]`,
		},
		{
			name:     "mixed content",
			input:    []string{"", "test", "with spaces"},
			expected: `["", "test", "with spaces"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatStringSlice(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "string value",
			input:    "test string",
			expected: `"test string"`,
		},
		{
			name:     "string with quotes",
			input:    `test "quoted" string`,
			expected: `"test \"quoted\" string"`,
		},
		{
			name:     "empty string",
			input:    "",
			expected: `""`,
		},
		{
			name:     "int value",
			input:    42,
			expected: "42",
		},
		{
			name:     "negative int",
			input:    -123,
			expected: "-123",
		},
		{
			name:     "int64 value",
			input:    int64(9223372036854775807),
			expected: "9223372036854775807",
		},
		{
			name:     "float64 value",
			input:    3.14159,
			expected: "3.14159",
		},
		{
			name:     "negative float",
			input:    -2.5,
			expected: "-2.5",
		},
		{
			name:     "zero float",
			input:    0.0,
			expected: "0",
		},
		{
			name:     "boolean true",
			input:    true,
			expected: "true",
		},
		{
			name:     "boolean false",
			input:    false,
			expected: "false",
		},
		{
			name:     "nil value",
			input:    nil,
			expected: "<nil>",
		},
		{
			name:     "slice value",
			input:    []string{"a", "b"},
			expected: "[a b]",
		},
		{
			name:     "map value",
			input:    map[string]int{"key": 123},
			expected: "map[key:123]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test the StaticFileServer function
func TestStaticFileServer(t *testing.T) {
	handler := StaticFileServer()
	assert.NotNil(t, handler)

	// Test that we can create the handler without errors
	// The actual file serving is tested by Go's http.FileServer tests

	// We can do a basic smoke test to ensure it doesn't panic
	req := httptest.NewRequest("GET", "/style.css", nil)
	w := httptest.NewRecorder()

	// This will return 404 since we don't have actual files, but shouldn't panic
	handler.ServeHTTP(w, req)
	// Just ensure it responds with some status (likely 404 for missing file)
	assert.True(t, w.Code >= 200 && w.Code < 600, "Handler should return a valid HTTP status code")
}

// Helper functions

func createValidUploadRequest(t *testing.T) *http.Request {
	return createUploadRequestWithParams(t, map[string]string{
		"iterations":    "5",
		"wait_temp":     "200",
		"wait_min":      "60",
		"extra_extrude": "0.1",
		"printer":       "test_printer",
	})
}

func createUploadRequestWithParams(t *testing.T, params map[string]string) *http.Request {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add form fields
	for key, value := range params {
		writer.WriteField(key, value)
	}

	// Add file only if not testing missing file
	if _, exists := params["no_file"]; !exists {
		part, err := writer.CreateFormFile("file", "test.txt")
		require.NoError(t, err)
		part.Write([]byte("test file content"))
	}

	writer.Close()

	req := httptest.NewRequest("POST", "/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}
