# Average Print Position Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `AveragePrintX` and `AveragePrintY` fields to `MarkerPositions` so printer templates can reference average print position.

**Architecture:** Extend the existing `extractGCodeCoordinates` function with sum/count accumulators for X and Y (tracked independently). Compute arithmetic mean at the end. Wire through to `MarkerPositions` struct.

**Tech Stack:** Go, existing processor package

---

### Task 1: Write failing test for average print position

**Files:**
- Modify: `internal/processor/processor_ProcessFile_test.go` (after line ~986)

**Step 1: Write the failing test**

Add a new test function after the existing `TestStreamingProcessor_findMarkerPositions_LastPrintCoordinates`:

```go
func TestStreamingProcessor_findMarkerPositions_AveragePrintCoordinates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		gcodeContent string
		expectedAvgX float64
		expectedAvgY float64
	}{
		{
			name: "single print command",
			gcodeContent: `M211 X0 Y0 Z0 ;turn off soft endstop
M1007 S1
G1 X10.0 Y20.0 E0.1
M625`,
			expectedAvgX: 10.0,
			expectedAvgY: 20.0,
		},
		{
			name: "two print commands",
			gcodeContent: `M211 X0 Y0 Z0 ;turn off soft endstop
M1007 S1
G1 X10.0 Y20.0 E0.1
G1 X30.0 Y40.0 E0.2
M625`,
			expectedAvgX: 20.0,
			expectedAvgY: 30.0,
		},
		{
			name: "three print commands",
			gcodeContent: `M211 X0 Y0 Z0 ;turn off soft endstop
M1007 S1
G1 X10.0 Y10.0 E0.1
G1 X20.0 Y20.0 E0.2
G1 X30.0 Y30.0 E0.3
M625`,
			expectedAvgX: 20.0,
			expectedAvgY: 20.0,
		},
		{
			name: "partial coordinates - X only second command",
			gcodeContent: `M211 X0 Y0 Z0 ;turn off soft endstop
M1007 S1
G1 X10.0 Y20.0 E0.1
G1 X30.0 E0.2
M625`,
			expectedAvgX: 20.0,
			expectedAvgY: 20.0,
		},
		{
			name: "partial coordinates - Y only second command",
			gcodeContent: `M211 X0 Y0 Z0 ;turn off soft endstop
M1007 S1
G1 X10.0 Y20.0 E0.1
G1 Y40.0 E0.2
M625`,
			expectedAvgX: 10.0,
			expectedAvgY: 30.0,
		},
		{
			name: "no print commands defaults to zero",
			gcodeContent: `M211 X0 Y0 Z0 ;turn off soft endstop
M1007 S1
G1 X10.0 Y20.0
M625`,
			expectedAvgX: 0.0,
			expectedAvgY: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpFile, err := os.CreateTemp(t.TempDir(), "gcode_test_*.gcode")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())
			defer tmpFile.Close()

			_, err = tmpFile.WriteString(tt.gcodeContent)
			if err != nil {
				t.Fatalf("Failed to write test content: %v", err)
			}

			tmpFile.Close()

			config := ProcessingRequest{
				Iterations: 1,
				Printer:    "unit-tests-gcode2",
			}

			processor, err := NewStreamingProcessor(config)
			if err != nil {
				t.Fatalf("Failed to create processor: %v", err)
			}

			positions, err := processor.findMarkerPositions(tmpFile.Name())
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if positions.AveragePrintX != tt.expectedAvgX {
				t.Errorf("AveragePrintX: expected %f, got %f", tt.expectedAvgX, positions.AveragePrintX)
			}

			if positions.AveragePrintY != tt.expectedAvgY {
				t.Errorf("AveragePrintY: expected %f, got %f", tt.expectedAvgY, positions.AveragePrintY)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -race -run TestStreamingProcessor_findMarkerPositions_AveragePrintCoordinates ./internal/processor/`
Expected: FAIL - `positions.AveragePrintX undefined (type *MarkerPositions has no field or method AveragePrintX)`

---

### Task 2: Add AveragePrintX/Y to MarkerPositions struct

**Files:**
- Modify: `internal/processor/processor.go:84-95`

**Step 1: Add fields to struct**

Add two fields after `LastPrintZ` (line 94):

```go
type MarkerPositions struct {
	EndInitSectionFirstLine  int64   // First line of start marker (0-based)
	EndInitSectionLastLine   int64   // Last line of start marker (0-based)
	EndPrintSectionFirstLine int64   // First line of end marker (0-based) - NEW
	EndPrintSectionLastLine  int64   // Last line of end marker (0-based) - UPDATED
	FirstPrintX              float64 // X coordinate from first print command (G1 with positive E) after marker
	FirstPrintY              float64 // Y coordinate from first print command (G1 with positive E) after marker
	FirstPrintZ              float64 // Z coordinate that was active during first print command after marker
	LastPrintX               float64 // X coordinate from last print command (G1 with positive E)
	LastPrintY               float64 // Y coordinate from last print command (G1 with positive E)
	LastPrintZ               float64 // Z coordinate that was active during last print command
	AveragePrintX            float64 // Average X coordinate across all print commands (G1 with positive E)
	AveragePrintY            float64 // Average Y coordinate across all print commands (G1 with positive E)
}
```

**Step 2: Run test to verify it still fails (different error now)**

Run: `go test -race -run TestStreamingProcessor_findMarkerPositions_AveragePrintCoordinates ./internal/processor/`
Expected: FAIL - tests compile but AveragePrintX/Y are 0.0 (not yet computed). The "two print commands" and "three print commands" cases should fail.

---

### Task 3: Implement average calculation in extractGCodeCoordinates

**Files:**
- Modify: `internal/processor/processor.go:385-493` (extractGCodeCoordinates function)
- Modify: `internal/processor/processor.go:362-379` (findMarkerPositions caller)

**Step 1: Update extractGCodeCoordinates signature and add accumulators**

Change the function signature (line 385) to return 8 values instead of 6:

```go
func (p *StreamingProcessor) extractGCodeCoordinates(filePath string, endInitSectionLastLine int64) (float64, float64, float64, float64, float64, float64, float64, float64, error) {
```

Update the error returns (lines 388, 455, 488) to return 8 zeros:

```go
return 0, 0, 0, 0, 0, 0, 0, 0, err
```

And line 488:
```go
return fx, fy, fz, lx, ly, lz, 0, 0, fmt.Errorf("no print commands found after end of init section at line %d", endInitSectionLastLine)
```

Add accumulators after the existing variable declarations (after line 397):

```go
var (
    firstPrintX, firstPrintY, firstPrintZ *float64
    lastPrintX, lastPrintY, lastPrintZ    *float64
    currentZ                              *float64
    firstPrintFound                       bool
    sumX, sumY                            float64
    countX, countY                        int
)
```

**Step 2: Accumulate in the print command block**

Inside the `if coords.E != nil && *coords.E > 0 && (coords.X != nil || coords.Y != nil)` block (after updating last print coords, around line 446), add:

```go
if coords.X != nil {
    sumX += *coords.X
    countX++
}

if coords.Y != nil {
    sumY += *coords.Y
    countY++
}
```

**Step 3: Compute averages and return**

Replace the final return (line 492) with:

```go
var avgX, avgY float64
if countX > 0 {
    avgX = sumX / float64(countX)
}
if countY > 0 {
    avgY = sumY / float64(countY)
}

return fx, fy, fz, lx, ly, lz, avgX, avgY, nil
```

**Step 4: Update findMarkerPositions caller**

Update line 363 to receive the extra values:

```go
firstPrintX, firstPrintY, firstPrintZ, lastPrintX, lastPrintY, lastPrintZ, avgPrintX, avgPrintY, err := p.extractGCodeCoordinates(filePath, initLast)
```

Update the struct initialization (lines 368-379) to include new fields:

```go
positions := &MarkerPositions{
    EndInitSectionFirstLine:  initFirst,
    EndInitSectionLastLine:   initLast,
    EndPrintSectionFirstLine: printFirst,
    EndPrintSectionLastLine:  printLast,
    FirstPrintX:              firstPrintX,
    FirstPrintY:              firstPrintY,
    FirstPrintZ:              firstPrintZ,
    LastPrintX:               lastPrintX,
    LastPrintY:               lastPrintY,
    LastPrintZ:               lastPrintZ,
    AveragePrintX:            avgPrintX,
    AveragePrintY:            avgPrintY,
}
```

**Step 5: Run tests to verify they pass**

Run: `go test -race -run TestStreamingProcessor_findMarkerPositions_AveragePrintCoordinates ./internal/processor/`
Expected: PASS

**Step 6: Run all tests to verify no regressions**

Run: `go test -race ./...`
Expected: All PASS

**Step 7: Run linter**

Run: `make lint`
Expected: PASS

**Step 8: Commit**

```bash
git add internal/processor/processor.go internal/processor/processor_ProcessFile_test.go
git commit -m "feat: add AveragePrintX and AveragePrintY to MarkerPositions"
```
