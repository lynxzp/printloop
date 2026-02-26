// file: internal/webserver/handlers_test.go
package webserver

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"printloop/internal/processor"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHomeHandler(t *testing.T) {
	// Initialize translations for tests
	err := LoadTranslations()
	require.NoError(t, err)

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
				t.Helper()
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
				t.Helper()
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
	t.Helper()
	// Setup test directories
	setupTestDirs := func(t *testing.T) {
		t.Helper()

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
			setupRequest: func(_ *testing.T) *http.Request {
				req := httptest.NewRequest("POST", "/upload", strings.NewReader("invalid"))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

				return req
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				t.Helper()
				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
				assert.Contains(t, w.Body.String(), "upload_form_error")
			},
		},
		{
			name: "missing file",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				var buf bytes.Buffer

				writer := multipart.NewWriter(&buf)
				_ = writer.WriteField("iterations", "5")
				_ = writer.Close()

				req := httptest.NewRequest("POST", "/upload", &buf)
				req.Header.Set("Content-Type", writer.FormDataContentType())

				return req
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				t.Helper()
				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
				assert.Contains(t, w.Body.String(), "processing_error")
			},
		},
		{
			name: "invalid iterations",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createUploadRequestWithParams(t, map[string]string{
					"iterations": "invalid",
				})
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				t.Helper()
				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
				assert.Contains(t, w.Body.String(), "invalid_parameters")
			},
		},
		{
			name: "large file within limit",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				var buf bytes.Buffer

				writer := multipart.NewWriter(&buf)
				_ = writer.WriteField("iterations", "2")

				part, err := writer.CreateFormFile("file", "large.txt")
				require.NoError(t, err)
				// Write a moderately large file (1KB)
				largeContent := strings.Repeat("test data line\n", 64)
				_, _ = part.Write([]byte(largeContent))
				_ = writer.Close()

				req := httptest.NewRequest("POST", "/upload", &buf)
				req.Header.Set("Content-Type", writer.FormDataContentType())

				return req
			},
			expectedStatus: http.StatusInternalServerError, // Will fail in processor
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				t.Helper()
				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
				assert.Contains(t, w.Body.String(), "invalid_printer_name")
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
				t.Helper()

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
				t.Helper()
				assert.Equal(t, "application/octet-stream", w.Header().Get("Content-Type"))
				assert.Contains(t, w.Header().Get("Content-Disposition"), req.FileName)
				assert.Equal(t, "test content", w.Body.String())
			},
		},
		{
			name: "file not found",
			setupFile: func(t *testing.T) processor.ProcessingRequest {
				t.Helper()

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
				t.Helper()

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
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder, _ processor.ProcessingRequest) {
				t.Helper()
				assert.Empty(t, w.Body.String())
			},
		},
		{
			name: "special characters in filename",
			setupFile: func(t *testing.T) processor.ProcessingRequest {
				t.Helper()

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
				t.Helper()
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
				require.NoError(t, err)

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
	t.Parallel()
	setupTestDirs := func(t *testing.T) {
		t.Helper()

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
				t.Helper()
				return createValidUploadRequest(t)
			},
			expectedError: false,
			validateReq: func(t *testing.T, req processor.ProcessingRequest) {
				t.Helper()
				assert.Equal(t, int64(5), req.Iterations)
				assert.Equal(t, int64(200), req.WaitBedCooldownTemp)
				assert.Equal(t, int64(60), req.WaitMin)
				assert.InEpsilon(t, 0.1, req.ExtraExtrude, 0.00001)
				assert.Equal(t, "test_printer", req.Printer)
				assert.Contains(t, req.FileName, "test.txt")
			},
		},
		{
			name: "invalid iterations",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createUploadRequestWithParams(t, map[string]string{
					"iterations": "invalid",
				})
			},
			expectedError: true,
		},
		{
			name: "zero iterations",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createUploadRequestWithParams(t, map[string]string{
					"iterations": "0",
				})
			},
			expectedError: true,
		},
		{
			name: "invalid waitBedCooldownTemp",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createUploadRequestWithParams(t, map[string]string{
					"iterations":          "5",
					"waitBedCooldownTemp": "invalid",
				})
			},
			expectedError: true,
		},
		{
			name: "negative waitBedCooldownTemp",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createUploadRequestWithParams(t, map[string]string{
					"iterations":          "5",
					"waitBedCooldownTemp": "-1",
				})
			},
			expectedError: true,
		},
		{
			name: "invalid wait_min",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

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
				t.Helper()

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
				t.Helper()

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
				t.Helper()

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
				t.Helper()

				var buf bytes.Buffer

				writer := multipart.NewWriter(&buf)

				_ = writer.WriteField("iterations", "5")
				_ = writer.Close()

				req := httptest.NewRequest("POST", "/upload", &buf)
				req.Header.Set("Content-Type", writer.FormDataContentType())

				return req
			},
			expectedError: true,
		},
		{
			name: "empty optional fields",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createUploadRequestWithParams(t, map[string]string{
					"iterations":          "5",
					"waitBedCooldownTemp": "",
					"wait_min":            "",
					"extra_extrude":       "",
					"printer":             "",
				})
			},
			expectedError: false,
			validateReq: func(t *testing.T, req processor.ProcessingRequest) {
				t.Helper()
				assert.Equal(t, int64(5), req.Iterations)
				assert.Empty(t, req.WaitBedCooldownTemp)
				assert.Empty(t, req.WaitMin)
				assert.Empty(t, req.ExtraExtrude)
				assert.Empty(t, req.Printer)
			},
		},
		{
			name: "custom template with whitespace",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createUploadRequestWithParams(t, map[string]string{
					"iterations":      "5",
					"custom_template": "  \n  G1 X10 Y10  \n  G1 Z5  \n  ",
				})
			},
			expectedError: false,
			validateReq: func(t *testing.T, req processor.ProcessingRequest) {
				t.Helper()
				assert.Equal(t, "G1 X10 Y10  \n  G1 Z5", req.CustomTemplate)
			},
		},
		{
			name: "very large iterations",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createUploadRequestWithParams(t, map[string]string{
					"iterations": "9223372036854775807", // max int64, exceeds 10000 limit
				})
			},
			expectedError: true,
		},
		{
			name: "filename with special characters",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				var buf bytes.Buffer

				writer := multipart.NewWriter(&buf)
				_ = writer.WriteField("iterations", "5")

				part, err := writer.CreateFormFile("file", "test file with spaces & symbols.gcode")
				require.NoError(t, err)

				_, _ = part.Write([]byte("test content"))
				_ = writer.Close()

				req := httptest.NewRequest("POST", "/upload", &buf)
				req.Header.Set("Content-Type", writer.FormDataContentType())

				return req
			},
			expectedError: false,
			validateReq: func(t *testing.T, req processor.ProcessingRequest) {
				t.Helper()
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
	t.Parallel()
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
				t.Helper()
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
				t.Helper()
				assert.Equal(t, "Missing printer parameter\n", w.Body.String())
			},
		},
		{
			name:           "empty printer parameter",
			method:         "GET",
			queryParams:    "?printer=",
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				t.Helper()
				assert.Equal(t, "Missing printer parameter\n", w.Body.String())
			},
		},
		{
			name:           "nonexistent printer",
			method:         "GET",
			queryParams:    "?printer=nonexistent_printer",
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, w.Body.String(), "Printer not found")
			},
		},
		{
			name:           "printer name with spaces",
			method:         "GET",
			queryParams:    "?printer=Test%20Printer%20Name",
			expectedStatus: http.StatusNotFound, // Will normalize to test-printer-name and likely not exist
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, w.Body.String(), "Printer not found")
			},
		},
		{
			name:           "printer name case insensitive",
			method:         "GET",
			queryParams:    "?printer=TEST_PRINTER",
			expectedStatus: http.StatusNotFound, // Will normalize to test_printer and likely not exist
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, w.Body.String(), "Printer not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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

// Test the StaticFileServer function
func TestStaticFileServer(t *testing.T) {
	t.Parallel()

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
	t.Helper()

	return createUploadRequestWithParams(t, map[string]string{
		"iterations":          "5",
		"waitBedCooldownTemp": "200",
		"wait_min":            "60",
		"extra_extrude":       "0.1",
		"printer":             "test_printer",
	})
}

func createUploadRequestWithParams(t *testing.T, params map[string]string) *http.Request {
	t.Helper()

	var buf bytes.Buffer

	writer := multipart.NewWriter(&buf)
	// Add form fields
	for key, value := range params {
		_ = writer.WriteField(key, value)
	}

	// Add file only if not testing missing file
	if _, exists := params["no_file"]; !exists {
		part, err := writer.CreateFormFile("file", "test.txt")
		require.NoError(t, err)

		_, _ = part.Write([]byte("test file content"))
	}

	_ = writer.Close()

	req := httptest.NewRequest("POST", "/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req
}
