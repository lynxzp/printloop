# Тимчасові директорії на запит + суфікс `_xN` — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Кожен запит обробляється у власній тимчасовій директорії `files/job-*`, а користувач отримує файл з чистим ім'ям і суфіксом `_xN` перед розширенням (N — кількість ітерацій), без timestamp-префікса.

**Architecture:** `receiveRequest` створює директорію через `os.MkdirTemp("files", "job-")` і кладе туди файл під очищеним оригінальним ім'ям; `UploadHandler` пише результат у ту саму директорію під ім'ям `name_xN.ext` і прибирає все одним `os.RemoveAll`; `sendResponse` віддає файл з коректно екранованим `Content-Disposition` через `mime.FormatMediaType`. `main.go` створює лише `files/` і на старті видаляє залишки `files/job-*`.

**Tech Stack:** Go stdlib (`os`, `path`, `path/filepath`, `mime`), testify.

**Spec:** `docs/superpowers/specs/2026-07-18-temp-dir-file-naming-design.md`

**Примітка про `Content-Disposition`:** `mime.FormatMediaType` віддає token-імена БЕЗ лапок (`filename=model_x2.gcode`), імена з пробілами — в лапках, не-ASCII — як `filename*=utf-8''%…` (перевірено на Go цього репозиторію). Тому всі перевірки заголовка в тестах — через `mime.ParseMediaType`, а не через `assert.Contains` з лапками.

**ВАЖЛИВО — правило проєкту:** git commit НЕ робити. Замість commit — `git add` (стейджити) і переходити до наступної задачі. Комітить лише користувач (див. CLAUDE.md printloop).

---

## File Structure

| Файл | Дія | Відповідальність |
|---|---|---|
| `internal/webserver/filename.go` | створити | чисті хелпери: `sanitizeFileName`, `resultFileName` |
| `internal/webserver/filename_test.go` | створити | table-driven тести хелперів |
| `internal/webserver/handlers.go` | змінити | `receiveRequest` (workDir), `UploadHandler` (шляхи, cleanup), `sendResponse` (нова сигнатура, FormatMediaType) |
| `internal/webserver/handlers_test.go` | змінити | оновити під нові сигнатури + нові кейси |
| `main.go` | змінити | лише `files/`; `cleanupStaleWorkDirs()` |

Всі команди запускаються з кореня репозиторію `services/printloop`.

---

### Task 1: Хелпери імен файлів (`sanitizeFileName`, `resultFileName`)

**Files:**
- Create: `internal/webserver/filename.go`
- Test: `internal/webserver/filename_test.go`

- [ ] **Step 1: Написати failing-тести**

Створити `internal/webserver/filename_test.go`:

```go
package webserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeFileName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "plain name", input: "model.gcode", expected: "model.gcode"},
		{name: "unix path traversal", input: "../../etc/passwd", expected: "passwd"},
		{name: "windows full path", input: `C:\Users\bob\part.gcode`, expected: "part.gcode"},
		{name: "mixed separators", input: `..\..//model.gcode`, expected: "model.gcode"},
		{name: "empty name", input: "", expected: "input.gcode"},
		{name: "dot", input: ".", expected: "input.gcode"},
		{name: "double dot", input: "..", expected: "input.gcode"},
		{name: "trailing separator", input: "dir/", expected: "input.gcode"},
		{name: "spaces and symbols kept", input: "test file & symbols.gcode", expected: "test file & symbols.gcode"},
		{name: "cyrillic kept", input: "модель.gcode", expected: "модель.gcode"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, sanitizeFileName(tt.input))
		})
	}
}

func TestResultFileName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		iterations int64
		expected   string
	}{
		{name: "regular gcode", input: "model.gcode", iterations: 5, expected: "model_x5.gcode"},
		{name: "no extension", input: "model", iterations: 3, expected: "model_x3"},
		{name: "multiple dots", input: "my.model.gcode", iterations: 2, expected: "my.model_x2.gcode"},
		{name: "max iterations", input: "part.gcode", iterations: 10000, expected: "part_x10000.gcode"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, resultFileName(tt.input, tt.iterations))
		})
	}
}
```

- [ ] **Step 2: Переконатися, що тести падають**

Run: `go test -race ./internal/webserver/ -run 'TestSanitizeFileName|TestResultFileName' -v`
Expected: FAIL (compile error: `undefined: sanitizeFileName`, `undefined: resultFileName`)

- [ ] **Step 3: Реалізувати хелпери**

Створити `internal/webserver/filename.go`:

```go
package webserver

import (
	"fmt"
	"path"
	"strings"
)

const fallbackFileName = "input.gcode"

// sanitizeFileName reduces a client-supplied file name to a safe base name.
// Browsers may send a full path (Windows: `C:\dir\part.gcode`) and a crafted
// request may contain `../` segments, so everything up to the last path
// separator is dropped.
func sanitizeFileName(name string) string {
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}

	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return fallbackFileName
	}

	return name
}

// resultFileName inserts an _xN suffix before the extension:
// "model.gcode" with 5 iterations becomes "model_x5.gcode".
func resultFileName(name string, iterations int64) string {
	ext := path.Ext(name)
	base := strings.TrimSuffix(name, ext)

	return fmt.Sprintf("%s_x%d%s", base, iterations, ext)
}
```

- [ ] **Step 4: Переконатися, що тести проходять**

Run: `go test -race ./internal/webserver/ -run 'TestSanitizeFileName|TestResultFileName' -v`
Expected: PASS (усі підтести)

- [ ] **Step 5: Застейджити (БЕЗ commit)**

```bash
git add internal/webserver/filename.go internal/webserver/filename_test.go
```

---

### Task 2: Переробити handlers.go (workDir, `_xN`, FormatMediaType)

`receiveRequest`, `UploadHandler` і `sendResponse` міняються разом — це один компілятивний юніт: зміна сигнатури `receiveRequest` ламає `UploadHandler`, а той викликає новий `sendResponse`.

**Files:**
- Modify: `internal/webserver/handlers.go` (UploadHandler:76-114, sendResponse:116-134, receiveRequest:136-215)
- Test: `internal/webserver/handlers_test.go`

- [ ] **Step 1: Оновити `TestReceiveRequest` під нову сигнатуру + нові кейси**

У `internal/webserver/handlers_test.go`:

1a. `setupTestDirs` всередині `TestReceiveRequest` (рядок ~323): замінити `os.MkdirAll("files/uploads", 0755)` на `os.MkdirAll("files", 0755)`:

```go
	setupTestDirs := func(t *testing.T) {
		t.Helper()

		err := os.MkdirAll("files", 0755)

		require.NoError(t, err)
		t.Cleanup(func() {
			os.RemoveAll("files")
		})
	}
```

1b. Цикл виконання (рядок ~549): нова сигнатура повертає `workDir`; на успіху перевіряємо директорію і файл у ній, на помилці — порожній `workDir`:

```go
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestDirs(t)

			req := tt.setupRequest(t)
			w := httptest.NewRecorder()

			result, workDir, err := receiveRequest(w, req)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Empty(t, workDir)
			} else {
				assert.NoError(t, err)
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
```

1c. Кейс "valid request" (рядок ~354): `assert.Contains(t, req.FileName, "test.txt")` → точна перевірка без timestamp-префікса:

```go
				assert.Equal(t, "test.txt", req.FileName)
```

1d. Кейс "filename with special characters" (рядок ~521): тіло `setupRequest` замінити викликом нового хелпера (див. 1f), а перевірку зробити точною:

```go
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
```

1e. Додати нові кейси в той самий слайс `tests` (після "filename with special characters"):

```go
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
			name: "dot dot filename falls back to default name",
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
```

1f. Додати хелпер поруч із `createUploadRequestWithParams` (кінець файлу):

```go
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
```

- [ ] **Step 2: Переписати `TestSendResponse` під нову сигнатуру**

Нова сигнатура: `sendResponse(w, filePath, downloadName)` — файл лежить за довільним шляхом, ім'я для користувача передається окремо. Тест більше не потребує `files/` — використовує `t.TempDir()`. Повністю замінити функцію `TestSendResponse` (рядки 207-319):

```go
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
```

Додати до імпортів `handlers_test.go`: `"mime"` та `"path/filepath"` (filepath — для Step 3).

- [ ] **Step 3: Оновити `TestUploadHandler`: нові директорії, e2e-успіх, перевірка cleanup**

3a. `setupTestDirs` всередині `TestUploadHandler` (рядок ~83): створювати лише `files`:

```go
	setupTestDirs := func(t *testing.T) {
		t.Helper()

		err := os.MkdirAll("files", 0755)

		require.NoError(t, err)
		t.Cleanup(func() {
			os.RemoveAll("files")
		})
	}
```

3b. Додати e2e-кейс успішної обробки у слайс `tests` (принтер `unit-tests` вбудований у бінарник через go:embed, маркери START_PRINT/END_PRINT — див. `internal/processor/printers/unit-tests.toml`):

```go
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
```

3c. У циклі виконання `TestUploadHandler` (після `checkResponse`) додати перевірку, що робочі директорії прибрані і на успіху, і на помилці:

```go
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
```

- [ ] **Step 4: Переконатися, що тести не компілюються / падають**

Run: `go test -race ./internal/webserver/ 2>&1 | head -20`
Expected: FAIL — compile errors (`receiveRequest` повертає 2 значення, а тест чекає 3; `sendResponse` викликається з 3 аргументами)

- [ ] **Step 5: Реалізувати зміни в `handlers.go`**

5a. Імпорти: додати `"mime"`, прибрати `"time"` (стане невикористаним).

5b. `UploadHandler` (рядки 76-114) — тіло після `receiveRequest`:

```go
func UploadHandler(w http.ResponseWriter, r *http.Request) {
	log := slog.With("handler", "UploadHandler")
	log.Info("Received upload request", "remote_addr", r.RemoteAddr)

	// Determine language for error messages
	lang := GetLanguageFromRequest(r)

	req, workDir, err := receiveRequest(w, r)
	if err != nil {
		log.Error("Failed to receive request", "error", err)
		WriteErrorResponseWithLang(w, err, http.StatusBadRequest, lang)

		return
	}

	defer func() {
		if rmErr := os.RemoveAll(workDir); rmErr != nil {
			log.Error("Failed to remove work directory", "dir", workDir, "error", rmErr)
		}
	}()

	inFileName := path.Join(workDir, req.FileName)
	resultName := resultFileName(req.FileName, req.Iterations)
	outFileName := path.Join(workDir, resultName)

	err = processor.ProcessFile(inFileName, outFileName, req)
	if err != nil {
		log.Error("Request processing failed", "error", err)
		WriteErrorResponseWithLang(w, err, http.StatusInternalServerError, lang)

		return
	}

	err = sendResponse(w, outFileName, resultName)
	if err != nil {
		log.Error("Failed to send response", "error", err)
		WriteErrorResponseWithLang(w, err, http.StatusInternalServerError, lang)

		return
	}

	log.Info("Request processed", "filename", resultName)
}
```

5c. `sendResponse` (рядки 116-134) — повна заміна:

```go
func sendResponse(w http.ResponseWriter, filePath, downloadName string) error {
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": downloadName})
	if disposition == "" {
		// FormatMediaType returns "" for values it cannot encode; fall back to a bare attachment
		disposition = "attachment"
	}

	w.Header().Set("Content-Disposition", disposition)
	w.Header().Set("Content-Type", "application/octet-stream")

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open result file %s: %w", filePath, err)
	}
	defer file.Close()

	_, err = io.Copy(w, file)
	if err != nil {
		return fmt.Errorf("failed writing response: %w", err)
	}

	return nil
}
```

5d. `receiveRequest` (рядки 136-215): сигнатура `(processor.ProcessingRequest, string, error)` — другим значенням workDir; на будь-якій помилці workDir порожній (внутрішній cleanup). Усі наявні `return req, fmt.Errorf(...)` / `return req, errors.New(...)` у валідації параметрів стають `return req, "", fmt.Errorf(...)` / `return req, "", errors.New(...)`. Хвіст функції (від `timestamp := ...` до кінця, рядки 198-214) замінити на:

```go
	req.FileName = sanitizeFileName(header.Filename)

	workDir, err := os.MkdirTemp("files", "job-")
	if err != nil {
		return req, "", fmt.Errorf("failed to create work directory: %w", err)
	}

	dst, err := os.Create(path.Join(workDir, req.FileName))
	if err != nil {
		// best-effort cleanup; the request already failed
		_ = os.RemoveAll(workDir)
		return req, "", fmt.Errorf("file creation failed: %w", err)
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		// best-effort cleanup; the request already failed
		_ = os.RemoveAll(workDir)
		return req, "", fmt.Errorf("file saving error: %w", err)
	}

	return req, workDir, nil
}
```

Повний перелік ранніх return-ів у `receiveRequest`, які отримують другий `""`:
- `form parsing error`
- `invalid iterations value`
- `invalid wait_temp value`
- `bed cooldown temperature must be at least 40°C...`
- `invalid wait_min value`
- `invalid extra_extrude value`
- `file retrieval error`

- [ ] **Step 6: Переконатися, що всі тести пакета проходять**

Run: `go test -race ./internal/webserver/ -v 2>&1 | tail -30`
Expected: PASS (включно з новими кейсами `TestReceiveRequest`, `TestSendResponse`, e2e-кейсом `TestUploadHandler`)

- [ ] **Step 7: Застейджити (БЕЗ commit)**

```bash
git add internal/webserver/handlers.go internal/webserver/handlers_test.go
```

---

### Task 3: main.go — лише `files/` + чистка залишків на старті

**Files:**
- Modify: `main.go:22-38` (створення директорій) + нова функція внизу файлу

- [ ] **Step 1: Замінити створення директорій**

У `main()` замінити три блоки `os.MkdirAll` (рядки 22-38) на:

```go
	err = os.MkdirAll("files", 0755)
	if err != nil {
		slog.Error("Failed to create files directory:", "err", err)
		return
	}

	cleanupStaleWorkDirs()
```

- [ ] **Step 2: Додати функцію чистки**

Додати в кінець `main.go` (після `initLogger`) і до імпортів додати `"path/filepath"`:

```go
// cleanupStaleWorkDirs removes per-request work directories left behind by a
// previous run that crashed or was killed mid-request.
func cleanupStaleWorkDirs() {
	stale, err := filepath.Glob("files/job-*")
	if err != nil {
		slog.Warn("Failed to scan for stale work directories", "err", err)
		return
	}

	for _, dir := range stale {
		if err := os.RemoveAll(dir); err != nil {
			slog.Warn("Failed to remove stale work directory", "dir", dir, "err", err)
		}
	}
}
```

- [ ] **Step 3: Перевірити компіляцію**

Run: `go build ./...`
Expected: успіх, без виводу

- [ ] **Step 4: Застейджити (БЕЗ commit)**

```bash
git add main.go
```

---

### Task 4: Повна верифікація

**Files:** нічого нового — тільки перевірки.

- [ ] **Step 1: Всі тести з race detector**

Run: `make test` (= `go test -race ./...`)
Expected: `ok` для всіх пакетів, без FAIL

- [ ] **Step 2: Лінтер**

Run: `make lint` (= `golangci-lint run`)
Expected: без зауважень. Якщо лінтер скаржиться на щось із цього плану — виправити і повторити Step 1.

- [ ] **Step 3: Переконатися, що все застейджено**

```bash
git status --short
```
Expected: усі змінені файли в індексі (`A`/`M` у першій колонці), незастейджених змін немає. Commit НЕ робити — комітить користувач.
