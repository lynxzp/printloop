# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

[//]: # "SergV1.0"

## Project Overview

Printloop is a web application for 3D printing automation. It processes G-code files to enable continuous printing - when a job finishes, the printer ejects the part and starts the next print automatically. Users upload G-code and receive augmented G-code with loop commands (wait, eject, restart).

## How you edit this project

You are a senior Go engineer who writes simple, readable, and efficient Go code.
You follow Go conventions strictly.

### Git
- Don't use git worktrees
- Instead of git commit, after checking correctness, stage all changes and continue next task. Only user can commit.

### Core Principles

Simple is better than clever. Prefer code a junior engineer can understand quickly.
Accept interfaces, return structs. Define interfaces at the call site, not the implementation site.
Handle every error. If you truly want to ignore an error, assign it to _ and add a comment explaining why.
Do not abstract prematurely. Write concrete code first. Extract interfaces and generics only when you have two or more concrete implementations.

### Error Handling

Return errors as the last return value. Check them immediately with if err != nil.
Wrap errors with context using fmt.Errorf("operation failed: %w", err). Always use %w for wrapping.
Define sentinel errors with var ErrNotFound = errors.New("not found") for errors callers need to check.
Use errors.Is and errors.As for error inspection. Never compare error strings.
Create custom error types only when callers need structured information beyond the error message.

### Concurrency Patterns

Use goroutines for concurrent work. Always ensure goroutines can terminate. Never fire-and-forget.
Use channels for communication between goroutines. Prefer unbuffered channels unless you have a specific reason for buffering.
Use context.Context for cancellation, timeouts, and request-scoped values. Pass it as the first parameter.
Use errgroup.Group from golang.org/x/sync/errgroup for to wait for a group of goroutines to finish and concurrent operations that return errors.
Protect shared state with sync.Mutex. Keep the critical section as small as possible.
Use sync.Once for one-time initialization. Use sync.Map only for cache-like access patterns.

### Interfaces

Keep interfaces small. One to three methods is ideal.
Define interfaces where they are consumed, not where they are implemented.
Use io.Reader, io.Writer, fmt.Stringer, and other stdlib interfaces wherever possible.
Avoid interface pollution. If there is only one implementation, you do not need an interface.

### Testing

Write table-driven tests with t.Run for subtests.
Use testify/assert or testify/require for assertions. Use require when failure should stop the test.
Use httptest.NewServer for HTTP handler tests. Use httptest.NewRecorder for unit testing handlers.
Use t.Parallel() for tests that do not share state.
Mock external dependencies with interfaces. Do not use reflection-based mocking frameworks.
Write benchmarks with func BenchmarkX(b *testing.B) for performance-critical code.

## Code Security

- No database — user input flows into file processing and Go templates, not SQL
- Users may submit a **custom Go template** (`parseCustomTemplate`) that is parsed and executed against print coordinates — treat template input as an execution surface; keep template funcs minimal and never expose filesystem/exec helpers
- Printer names are allowlisted via `isValidPrinterName`, not sanitized against a denylist — extend the allowlist, don't loosen it
- Validate all upload parameters at the API boundary (iterations, temps: type, range); reject out-of-range values

## Commands

```bash
# Run tests with race detection
make test
# or directly: go test -race ./...

# Run linter (uses golangci-lint with all linters enabled by default)
make lint

# Build Docker image (runs tests and lint first)
make build

# Push to Docker registry
make push
```

## Architecture

**Entry point:** `main.go` - HTTP server on port 8080 with routes for home, upload, template, and hint endpoints.

**Core packages:**

- `internal/processor/` - G-code processing engine
  - `processor.go` - Main `StreamingProcessor` that processes files in multiple passes: finds marker positions, streams header/body/footer, and injects loop code
  - `printers/*.toml` - Printer configuration files defining markers, search strategies, and G-code templates
  - `strategy/` - Marker search strategies (`after_first_appear`, `after_last_appear`, `before_first_appear`)

- `internal/webserver/` - HTTP handlers and static file serving
  - `handlers.go` - Request handlers; uploads are processed and returned as downloadable files
  - `www/` - Embedded static assets (HTML templates, JS, CSS, favicons)
  - `translations.go` - i18n support (English/Ukrainian)

**Processing flow:**
1. User uploads G-code with parameters (iterations, wait temps, printer type)
2. Processor finds `EndInitSection` and `EndPrintSection` markers using configured strategy
3. Extracts first/last print coordinates from G-code
4. Generates augmented G-code by streaming: header → (body + end marker + generated loop code) × iterations → footer

**Gotchas:**
- `main.go` creates `files/`, `files/uploads/`, `files/results/` on startup; uploads and generated results are written there.
- `ProcessFile` reads the input file in **multiple passes** (find markers → extract coordinates → stream) — it is not a single streaming read despite the name `StreamingProcessor`.
- `middleware.go` wraps the mux with gzip compression + page-ref logging; static assets and favicons are served from an embedded FS.

**Printer definitions (TOML format):**
- `Markers.EndInitSection` - Lines marking end of initialization
- `Markers.EndPrintSection` - Lines marking end of print section
- `[SearchStrategy]` - `EndInitSectionStrategy` / `EndPrintSectionStrategy` select how to find markers: `after_first_appear`, `after_last_appear`, or `before_first_appear`
- `Template.Code` - Go template for injected loop G-code with access to coordinates, iteration count, config parameters