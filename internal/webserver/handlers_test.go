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
	"path/filepath"
	"printloop/internal/processor"
	"strings"
	"testing"
)

func TestHomeHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		setupHTML      func(t *testing.T) string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "valid GET request",
			method: "GET",
			setupHTML: func(t *testing.T) string {
				content := "<html><body>Test HTML</body></html>"
				tmpDir := t.TempDir()
				htmlFile := filepath.Join(tmpDir, "index.html")
				err := os.WriteFile(htmlFile, []byte(content), 0644)
				require.NoError(t, err)

				// Create www directory and move file there
				wwwDir := "www"
				err = os.MkdirAll(wwwDir, 0755)
				require.NoError(t, err)
				t.Cleanup(func() { os.RemoveAll(wwwDir) })

				err = os.WriteFile(filepath.Join(wwwDir, "index.html"), []byte(content), 0644)
				require.NoError(t, err)

				return content
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "<html><body>Test HTML</body></html>",
		},
		{
			name:           "invalid method",
			method:         "POST",
			setupHTML:      func(t *testing.T) string { return "" },
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed\n",
		},
		{
			name:   "file read error",
			method: "GET",
			setupHTML: func(t *testing.T) string {
				// Don't create the file to trigger read error
				return ""
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Internal server error\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedContent := tt.setupHTML(t)

			req := httptest.NewRequest(tt.method, "/", nil)
			w := httptest.NewRecorder()

			HomeHandler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
				assert.Equal(t, expectedContent, w.Body.String())
			} else {
				assert.Equal(t, tt.expectedBody, w.Body.String())
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupFile(t)
			w := httptest.NewRecorder()

			err := sendResponse(w, req)

			if tt.expectedStatus == http.StatusOK {
				assert.NoError(t, err)
				assert.Equal(t, "application/octet-stream", w.Header().Get("Content-Type"))
				assert.Contains(t, w.Header().Get("Content-Disposition"), req.FileName)
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
