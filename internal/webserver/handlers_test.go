// file: internal/webserver/handlers_test.go
package webserver

import (
	"bytes"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
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
			req := httptest.NewRequestWithContext(t.Context(), tt.method, "/", nil)
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
		t.Helper()

		err := os.MkdirAll("files", 0755)

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
				t.Helper()

				req := httptest.NewRequestWithContext(t.Context(), "POST", "/upload", strings.NewReader("invalid"))
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

				req := httptest.NewRequestWithContext(t.Context(), "POST", "/upload", &buf)
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

				req := httptest.NewRequestWithContext(t.Context(), "POST", "/upload", &buf)
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
		{
			name: "successful processing returns file with _xN suffix",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				var buf bytes.Buffer

				writer := multipart.NewWriter(&buf)
				_ = writer.WriteField("iterations", "2")
				_ = writer.WriteField("printer", "unit-tests")

				part, err := writer.CreateFormFile("file", "model.gcode")
				require.NoError(t, err)

				_, _ = part.Write([]byte("HEADER\nSTART_PRINT\nBODY\nEND_PRINT\nFOOTER\n"))
				_ = writer.Close()

				req := httptest.NewRequestWithContext(t.Context(), "POST", "/upload", &buf)
				req.Header.Set("Content-Type", writer.FormDataContentType())

				return req
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				t.Helper()

				mediaType, params, err := mime.ParseMediaType(w.Header().Get("Content-Disposition"))
				require.NoError(t, err)
				assert.Equal(t, "attachment", mediaType)
				assert.Equal(t, "model_x2.gcode", params["filename"])
				assert.Contains(t, w.Body.String(), "; Generated code - Iteration 2")
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

			leftovers, err := filepath.Glob("files/job-*")
			require.NoError(t, err)
			assert.Empty(t, leftovers, "work directories must be removed after the request")
		})
	}
}

func TestSendResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setupFile     func(t *testing.T) (filePath, downloadName string)
		expectedError bool
		checkResponse func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "valid file send",
			setupFile: func(t *testing.T) (string, string) {
				t.Helper()

				filePath := path.Join(t.TempDir(), "result.gcode")
				err := os.WriteFile(filePath, []byte("test content"), 0644)
				require.NoError(t, err)

				return filePath, "test_file_x5.gcode"
			},
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				t.Helper()
				assert.Equal(t, "application/octet-stream", w.Header().Get("Content-Type"))

				mediaType, params, err := mime.ParseMediaType(w.Header().Get("Content-Disposition"))
				require.NoError(t, err)
				assert.Equal(t, "attachment", mediaType)
				assert.Equal(t, "test_file_x5.gcode", params["filename"])
				assert.Equal(t, "test content", w.Body.String())
			},
		},
		{
			name: "file not found",
			setupFile: func(t *testing.T) (string, string) {
				t.Helper()
				return path.Join(t.TempDir(), "nonexistent.gcode"), "nonexistent.gcode"
			},
			expectedError: true,
		},
		{
			name: "empty file",
			setupFile: func(t *testing.T) (string, string) {
				t.Helper()

				filePath := path.Join(t.TempDir(), "empty.gcode")
				err := os.WriteFile(filePath, []byte(""), 0644)
				require.NoError(t, err)

				return filePath, "empty_x2.gcode"
			},
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				t.Helper()
				assert.Empty(t, w.Body.String())
			},
		},
		{
			name: "special characters in download name",
			setupFile: func(t *testing.T) (string, string) {
				t.Helper()

				filePath := path.Join(t.TempDir(), "result.gcode")
				err := os.WriteFile(filePath, []byte("special content"), 0644)
				require.NoError(t, err)

				return filePath, `test "quoted" & symbols_x5.gcode`
			},
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				t.Helper()

				mediaType, params, err := mime.ParseMediaType(w.Header().Get("Content-Disposition"))
				require.NoError(t, err)
				assert.Equal(t, "attachment", mediaType)
				assert.Equal(t, `test "quoted" & symbols_x5.gcode`, params["filename"])
				assert.Equal(t, "special content", w.Body.String())
			},
		},
		{
			name: "non-ascii download name",
			setupFile: func(t *testing.T) (string, string) {
				t.Helper()

				filePath := path.Join(t.TempDir(), "result.gcode")
				err := os.WriteFile(filePath, []byte("cyrillic content"), 0644)
				require.NoError(t, err)

				return filePath, "модель_x5.gcode"
			},
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				t.Helper()

				mediaType, params, err := mime.ParseMediaType(w.Header().Get("Content-Disposition"))
				require.NoError(t, err)
				assert.Equal(t, "attachment", mediaType)
				assert.Equal(t, "модель_x5.gcode", params["filename"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			filePath, downloadName := tt.setupFile(t)
			w := httptest.NewRecorder()

			err := sendResponse(w, filePath, downloadName)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)

				if tt.checkResponse != nil {
					tt.checkResponse(t, w)
				}
			}
		})
	}
}

func TestReceiveRequest(t *testing.T) {
	t.Parallel()
	setupTestDirs := func(t *testing.T) {
		t.Helper()

		err := os.MkdirAll("files", 0755)

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
				assert.Equal(t, "test.txt", req.FileName)
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

				req := httptest.NewRequestWithContext(t.Context(), "POST", "/upload", &buf)
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
				return createUploadRequestWithFileName(t, "test file with spaces & symbols.gcode")
			},
			expectedError: false,
			validateReq: func(t *testing.T, req processor.ProcessingRequest) {
				t.Helper()
				assert.Equal(t, "test file with spaces & symbols.gcode", req.FileName)
			},
		},
		{
			name: "path traversal filename is reduced to base name",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createUploadRequestWithFileName(t, "../../evil.gcode")
			},
			expectedError: false,
			validateReq: func(t *testing.T, req processor.ProcessingRequest) {
				t.Helper()
				assert.Equal(t, "evil.gcode", req.FileName)
			},
		},
		{
			name: "windows full path filename is reduced to base name",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createUploadRequestWithFileName(t, `C:\Users\bob\part.gcode`)
			},
			expectedError: false,
			validateReq: func(t *testing.T, req processor.ProcessingRequest) {
				t.Helper()
				assert.Equal(t, "part.gcode", req.FileName)
			},
		},
		{
			name: "double dot filename falls back to default name",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()
				return createUploadRequestWithFileName(t, "..")
			},
			expectedError: false,
			validateReq: func(t *testing.T, req processor.ProcessingRequest) {
				t.Helper()
				assert.Equal(t, "input.gcode", req.FileName)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestDirs(t)

			req := tt.setupRequest(t)
			w := httptest.NewRecorder()

			result, workDir, err := receiveRequest(w, req)

			if tt.expectedError {
				require.Error(t, err)
				assert.Empty(t, workDir)
			} else {
				require.NoError(t, err)
				assert.True(t, strings.HasPrefix(workDir, path.Join("files", "job-")),
					"workDir must be created inside files/: %s", workDir)
				assert.DirExists(t, workDir)
				assert.FileExists(t, path.Join(workDir, result.FileName))

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

			req := httptest.NewRequestWithContext(t.Context(), tt.method, "/template"+tt.queryParams, nil)
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
	req := httptest.NewRequestWithContext(t.Context(), "GET", "/style.css", nil)
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

	req := httptest.NewRequestWithContext(t.Context(), "POST", "/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req
}

func createUploadRequestWithFileName(t *testing.T, fileName string) *http.Request {
	t.Helper()

	var buf bytes.Buffer

	writer := multipart.NewWriter(&buf)
	_ = writer.WriteField("iterations", "5")

	part, err := writer.CreateFormFile("file", fileName)
	require.NoError(t, err)

	_, _ = part.Write([]byte("test content"))
	_ = writer.Close()

	req := httptest.NewRequestWithContext(t.Context(), "POST", "/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req
}
