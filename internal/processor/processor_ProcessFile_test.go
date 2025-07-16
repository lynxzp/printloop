// file: internal/processor/processor_ProcessFile_test.go
package processor

import (
	"bufio"
	"os"
	"path/filepath"
	"printloop/internal/types"
	"strings"
	"testing"
)

// Test core logic with simple string slices (no I/O) using the new streaming processor
func TestStreamingProcessor_ProcessStream(t *testing.T) {
	tests := []struct {
		name        string
		input       []string
		markers     PositionMarkers
		expected    []string
		expectError bool
	}{
		{
			name: "single line start marker - sample 1",
			input: []string{
				"HEADER1",
				"HEADER2",
				"START_PRINT",
				"PRINT_LINE1",
				"PRINT_LINE2",
				"END_PRINT",
				"FOOTER1",
			},
			markers: PositionMarkers{
				StartMarker: []string{"START_PRINT"},
				EndMarker:   "END_PRINT",
			},
			expected: []string{
				"HEADER1",
				"HEADER2",
				"START_PRINT",
				"PRINT_LINE1",
				"PRINT_LINE2",
				"END_PRINT",
				"; Generated code - Iteration 1",
				"; Generated code - End iteration 1",
				"PRINT_LINE1",
				"PRINT_LINE2",
				"END_PRINT",
				"; Generated code - Iteration 2",
				"; Generated code - End iteration 2",
				"FOOTER1",
			},
		},
		{
			name: "single line start marker - sample 2",
			input: []string{
				"HEADER",
				"START_PRINT",
				"BODY",
				"END_PRINT",
				"FOOTER",
			},
			markers: PositionMarkers{
				StartMarker: []string{"START_PRINT"},
				EndMarker:   "END_PRINT",
			},
			expected: []string{
				"HEADER",
				"START_PRINT",
				"BODY",
				"END_PRINT",
				"; Generated code - Iteration 1",
				"; Generated code - End iteration 1",
				"BODY",
				"END_PRINT",
				"; Generated code - Iteration 2",
				"; Generated code - End iteration 2",
				"FOOTER",
			},
		},
		{
			name: "multiline start marker - 2 lines",
			input: []string{
				"HEADER1",
				"HEADER2",
				"START_PRINT_LINE1",
				"START_PRINT_LINE2",
				"PRINT_LINE1",
				"PRINT_LINE2",
				"END_PRINT",
				"FOOTER1",
			},
			markers: PositionMarkers{
				StartMarker: []string{"START_PRINT_LINE1", "START_PRINT_LINE2"},
				EndMarker:   "END_PRINT",
			},
			expected: []string{
				"HEADER1",
				"HEADER2",
				"START_PRINT_LINE1",
				"START_PRINT_LINE2",
				"PRINT_LINE1",
				"PRINT_LINE2",
				"END_PRINT",
				"; Generated code - Iteration 1",
				"; Generated code - End iteration 1",
				"PRINT_LINE1",
				"PRINT_LINE2",
				"END_PRINT",
				"; Generated code - Iteration 2",
				"; Generated code - End iteration 2",
				"FOOTER1",
			},
		},
		{
			name: "multiline start marker - 3 lines",
			input: []string{
				"HEADER",
				"M1007 S1",
				"G1 X0 Y0",
				"G1 Z0.2",
				"BODY_LINE1",
				"BODY_LINE2",
				"G625",
				"FOOTER",
			},
			markers: PositionMarkers{
				StartMarker: []string{"M1007 S1", "G1 X0 Y0", "G1 Z0.2"},
				EndMarker:   "G625",
			},
			expected: []string{
				"HEADER",
				"M1007 S1",
				"G1 X0 Y0",
				"G1 Z0.2",
				"BODY_LINE1",
				"BODY_LINE2",
				"G625",
				"; Generated code - Iteration 1",
				"; Generated code - End iteration 1",
				"BODY_LINE1",
				"BODY_LINE2",
				"G625",
				"; Generated code - Iteration 2",
				"; Generated code - End iteration 2",
				"FOOTER",
			},
		},
		{
			name: "multi end with multiline start",
			input: []string{
				"HEADER",
				"START1",
				"START2",
				"BODY",
				"END_PRINT",
				"BODY",
				"END_PRINT",
				"FOOTER",
			},
			markers: PositionMarkers{
				StartMarker: []string{"START1", "START2"},
				EndMarker:   "END_PRINT",
			},
			expected: []string{
				"HEADER",
				"START1",
				"START2",
				"BODY",
				"END_PRINT",
				"BODY",
				"END_PRINT",
				"; Generated code - Iteration 1",
				"; Generated code - End iteration 1",
				"BODY",
				"END_PRINT",
				"BODY",
				"END_PRINT",
				"; Generated code - Iteration 2",
				"; Generated code - End iteration 2",
				"FOOTER",
			},
		},
		{
			name: "empty lines in multiline start marker",
			input: []string{
				"HEADER",
				"START1",
				" ",
				"START2",
				"BODY",
				"END_PRINT",
				"BODY",
				"END_PRINT",
				"FOOTER",
			},
			markers: PositionMarkers{
				StartMarker: []string{"START1", "START2"},
				EndMarker:   "END_PRINT",
			},
			expected: []string{
				"HEADER",
				"START1",
				" ",
				"START2",
				"BODY",
				"END_PRINT",
				"BODY",
				"END_PRINT",
				"; Generated code - Iteration 1",
				"; Generated code - End iteration 1",
				"BODY",
				"END_PRINT",
				"BODY",
				"END_PRINT",
				"; Generated code - Iteration 2",
				"; Generated code - End iteration 2",
				"FOOTER",
			},
		},
		{
			name: "comment in multiline start marker",
			input: []string{
				"HEADER",
				"START1",
				"; This is a comment",
				"START2",
				"BODY",
				"END_PRINT",
				"BODY",
				"END_PRINT",
				"FOOTER",
			},
			markers: PositionMarkers{
				StartMarker: []string{"START1", "START2"},
				EndMarker:   "END_PRINT",
			},
			expected: []string{
				"HEADER",
				"START1",
				"; This is a comment",
				"START2",
				"BODY",
				"END_PRINT",
				"BODY",
				"END_PRINT",
				"; Generated code - Iteration 1",
				"; Generated code - End iteration 1",
				"BODY",
				"END_PRINT",
				"BODY",
				"END_PRINT",
				"; Generated code - Iteration 2",
				"; Generated code - End iteration 2",
				"FOOTER",
			},
		},
		{
			name: "comment directly in line",
			input: []string{
				"HEADER",
				"START1; This is a comment",
				"START2",
				"BODY",
				"END_PRINT",
				"BODY",
				"END_PRINT",
				"FOOTER",
			},
			markers: PositionMarkers{
				StartMarker: []string{"START1", "START2"},
				EndMarker:   "END_PRINT",
			},
			expected: []string{
				"HEADER",
				"START1", "; This is a comment",
				"START2",
				"BODY",
				"END_PRINT",
				"BODY",
				"END_PRINT",
				"; Generated code - Iteration 1",
				"; Generated code - End iteration 1",
				"BODY",
				"END_PRINT",
				"BODY",
				"END_PRINT",
				"; Generated code - Iteration 2",
				"; Generated code - End iteration 2",
				"FOOTER",
			},
		},
		{
			name: "spaces in start marker",
			input: []string{
				"HEADER",
				" START1 ",
				"START2",
				"BODY",
				"END_PRINT",
				"BODY",
				"END_PRINT",
				"FOOTER",
			},
			markers: PositionMarkers{
				StartMarker: []string{"START1", "START2"},
				EndMarker:   "END_PRINT",
			},
			expected: []string{
				"HEADER",
				" START1 ",
				"START2",
				"BODY",
				"END_PRINT",
				"BODY",
				"END_PRINT",
				"; Generated code - Iteration 1",
				"; Generated code - End iteration 1",
				"BODY",
				"END_PRINT",
				"BODY",
				"END_PRINT",
				"; Generated code - Iteration 2",
				"; Generated code - End iteration 2",
				"FOOTER",
			},
		},
		{
			name: "spaces in end marker",
			input: []string{
				"HEADER",
				"START1",
				"START2",
				"BODY",
				"END_PRINT ",
				"BODY",
				" END_PRINT",
				"FOOTER",
			},
			markers: PositionMarkers{
				StartMarker: []string{"START1", "START2"},
				EndMarker:   "END_PRINT",
			},
			expected: []string{
				"HEADER",
				"START1",
				"START2",
				"BODY",
				"END_PRINT ",
				"BODY",
				" END_PRINT",
				"; Generated code - Iteration 1",
				"; Generated code - End iteration 1",
				"BODY",
				"END_PRINT ",
				"BODY",
				" END_PRINT",
				"; Generated code - Iteration 2",
				"; Generated code - End iteration 2",
				"FOOTER",
			},
		},
		{
			name: "comments in end marker",
			input: []string{
				"HEADER",
				"START1",
				"START2",
				"BODY",
				"END_PRINT ; This is an end comment",
				"BODY",
				" END_PRINT ; Another end comment",
				"FOOTER",
			},
			markers: PositionMarkers{
				StartMarker: []string{"START1", "START2"},
				EndMarker:   "END_PRINT",
			},
			expected: []string{
				"HEADER",
				"START1",
				"START2",
				"BODY",
				"END_PRINT ; This is an end comment",
				"BODY",
				" END_PRINT ; Another end comment",
				"; Generated code - Iteration 1",
				"; Generated code - End iteration 1",
				"BODY",
				"END_PRINT ; This is an end comment",
				"BODY",
				" END_PRINT ; Another end comment",
				"; Generated code - Iteration 2",
				"; Generated code - End iteration 2",
				"FOOTER",
			},
		},
		{
			name:  "missing start marker - multiline",
			input: []string{"HEADER", "START1", "BODY", "END_PRINT"},
			markers: PositionMarkers{
				StartMarker: []string{"START1", "START2"},
				EndMarker:   "END_PRINT",
			},
			expectError: true,
		},
		{
			name:  "missing end marker",
			input: []string{"HEADER", "START1", "START2", "BODY"},
			markers: PositionMarkers{
				StartMarker: []string{"START1", "START2"},
				EndMarker:   "END_PRINT",
			},
			expectError: true,
		},
		{
			name:  "empty start marker",
			input: []string{"HEADER", "BODY", "END_PRINT"},
			markers: PositionMarkers{
				StartMarker: []string{},
				EndMarker:   "END_PRINT",
			},
			expectError: true,
		},
		{
			name: "partial multiline start marker match",
			input: []string{
				"HEADER",
				"START1",
				"WRONG_LINE",
				"START2",
				"BODY",
				"END_PRINT",
				"FOOTER",
			},
			markers: PositionMarkers{
				StartMarker: []string{"START1", "START2"},
				EndMarker:   "END_PRINT",
			},
			expectError: true,
		},
		{
			name: "long text line",
			input: []string{
				"HEADER1 ;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;",
				"HEADER2",
				"START_PRINT",
				"PRINT_LINE1",
				"PRINT_LINE2",
				"END_PRINT",
				"FOOTER1",
			},
			markers: PositionMarkers{
				StartMarker: []string{"START_PRINT"},
				EndMarker:   "END_PRINT",
			},
			expected: []string{
				"HEADER1 ;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;",
				"HEADER2",
				"START_PRINT",
				"PRINT_LINE1",
				"PRINT_LINE2",
				"END_PRINT",
				"; Generated code - Iteration 1",
				"; Generated code - End iteration 1",
				"PRINT_LINE1",
				"PRINT_LINE2",
				"END_PRINT",
				"; Generated code - Iteration 2",
				"; Generated code - End iteration 2",
				"FOOTER1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir := t.TempDir()
			inputPath := filepath.Join(tempDir, "input.txt")
			outputPath := filepath.Join(tempDir, "output.txt")

			// Write input file
			if err := writeLinesToFile(inputPath, tt.input); err != nil {
				t.Fatalf("Failed to write input file: %v", err)
			}

			// Create processor with test markers
			config := types.ProcessingRequest{
				Iterations: 2, // Based on expected outputs showing 2 iterations
			}
			processor := NewStreamingProcessor(config, tt.markers)

			// Process the file
			err := processor.ProcessFile(inputPath, outputPath)

			// Check error expectation
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Read output file
			actualOutput, err := readLinesFromFile(outputPath)
			if err != nil {
				t.Fatalf("Failed to read output file: %v", err)
			}

			// Compare with expected output
			if !equalStringSlices(actualOutput, tt.expected) {
				t.Errorf("Output mismatch\nExpected:\n%s\nActual:\n%s",
					strings.Join(tt.expected, "\n"),
					strings.Join(actualOutput, "\n"))
			}
		})
	}
}

// Helper function to write lines to file
func writeLinesToFile(filePath string, lines []string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	for _, line := range lines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return err
		}
	}

	return nil
}

// Helper function to read lines from file
func readLinesFromFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}

// Helper function to compare string slices
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
