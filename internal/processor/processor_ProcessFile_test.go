// file: internal/processor/processor_ProcessFile_test.go
package processor

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test core logic with simple string slices (no I/O) using the new streaming processor
func TestStreamingProcessor_ProcessStream(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		input       []string
		printerName string
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
			printerName: "unit-tests",
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
			printerName: "unit-tests",
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
			printerName: "unit-tests-multiline",
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
			printerName: "unit-tests-gcode",
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
				"START_PRINT",
				"BODY",
				"END_PRINT",
				"BODY",
				"END_PRINT",
				"FOOTER",
			},
			printerName: "unit-tests",
			expected: []string{
				"HEADER",
				"START_PRINT",
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
				"START_PRINT_LINE1",
				" ",
				"START_PRINT_LINE2",
				"BODY",
				"END_PRINT",
				"BODY",
				"END_PRINT",
				"FOOTER",
			},
			printerName: "unit-tests-multiline",
			expected: []string{
				"HEADER",
				"START_PRINT_LINE1",
				" ",
				"START_PRINT_LINE2",
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
				"START_PRINT_LINE1",
				"; This is a comment",
				"START_PRINT_LINE2",
				"BODY",
				"END_PRINT",
				"BODY",
				"END_PRINT",
				"FOOTER",
			},
			printerName: "unit-tests-multiline",
			expected: []string{
				"HEADER",
				"START_PRINT_LINE1",
				"; This is a comment",
				"START_PRINT_LINE2",
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
				"START_PRINT; This is a comment",
				"BODY",
				"END_PRINT",
				"BODY",
				"END_PRINT",
				"FOOTER",
			},
			printerName: "unit-tests",
			expected: []string{
				"HEADER",
				"START_PRINT", "; This is a comment",
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
				" START_PRINT ",
				"BODY",
				"END_PRINT",
				"BODY",
				"END_PRINT",
				"FOOTER",
			},
			printerName: "unit-tests",
			expected: []string{
				"HEADER",
				" START_PRINT ",
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
				"START_PRINT",
				"BODY",
				"END_PRINT ",
				"BODY",
				" END_PRINT",
				"FOOTER",
			},
			printerName: "unit-tests",
			expected: []string{
				"HEADER",
				"START_PRINT",
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
				"START_PRINT",
				"BODY",
				"END_PRINT ; This is an end comment",
				"BODY",
				" END_PRINT ; Another end comment",
				"FOOTER",
			},
			printerName: "unit-tests",
			expected: []string{
				"HEADER",
				"START_PRINT",
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
			name:        "missing start marker - multiline",
			input:       []string{"HEADER", "START_PRINT_LINE1", "BODY", "END_PRINT"},
			printerName: "unit-tests-multiline",
			expectError: true,
		},
		{
			name:        "missing end marker",
			input:       []string{"HEADER", "START_PRINT", "BODY"},
			printerName: "unit-tests",
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
			printerName: "unit-tests-multiline",
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
			printerName: "unit-tests",
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
			t.Parallel()
			// Create temporary directory
			tempDir := t.TempDir()
			inputPath := filepath.Join(tempDir, "input.txt")
			outputPath := filepath.Join(tempDir, "output.txt")

			// Write input file
			if err := writeLinesToFile(inputPath, tt.input); err != nil {
				t.Fatalf("Failed to write input file: %v", err)
			}

			// Create processor with test configuration
			config := ProcessingRequest{
				Iterations: 2, // Based on expected outputs showing 2 iterations
				Printer:    tt.printerName,
			}
			processor, err := NewStreamingProcessor(config)
			if err != nil {
				if tt.expectError {
					// If we expect an error and got one during processor creation, that's also valid
					return
				}
				t.Fatalf("Failed to create processor: %v", err)
			}

			// Process the file
			err = processor.ProcessFile(inputPath, outputPath)

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

func TestStreamingProcessor_parseGCodeLine(t *testing.T) {
	t.Parallel()
	// Helper function to create float64 pointer
	floatPtr := func(f float64) *float64 { return &f }

	// Create a processor instance for testing
	config := ProcessingRequest{
		Iterations: 1,
		Printer:    "unit-tests",
	}
	p, err := NewStreamingProcessor(config)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected *GCodeCoordinates
	}{
		{
			name:  "G1 with all coordinates",
			input: "G1 X24.811 Y159.285 Z3.601 E0.01274",
			expected: &GCodeCoordinates{
				X: floatPtr(24.811),
				Y: floatPtr(159.285),
				Z: floatPtr(3.601),
				E: floatPtr(0.01274),
			},
		},
		{
			name:  "G1 with X and Y only",
			input: "G1 X32.183 Y151.913",
			expected: &GCodeCoordinates{
				X: floatPtr(32.183),
				Y: floatPtr(151.913),
				Z: nil,
				E: nil,
			},
		},
		{
			name:  "G1 with Z only",
			input: "G1 Z3.601",
			expected: &GCodeCoordinates{
				X: nil,
				Y: nil,
				Z: floatPtr(3.601),
				E: nil,
			},
		},
		{
			name:  "G1 with E only",
			input: "G1 E3.601",
			expected: &GCodeCoordinates{
				X: nil,
				Y: nil,
				Z: nil,
				E: floatPtr(3.601),
			},
		},
		{
			name:  "G1 with Y only",
			input: "G1 Y3.601",
			expected: &GCodeCoordinates{
				X: nil,
				Y: floatPtr(3.601),
				Z: nil,
				E: nil,
			},
		},
		{
			name:  "G1 with positive E",
			input: "G1 X24.811 Y159.285 E.01274",
			expected: &GCodeCoordinates{
				X: floatPtr(24.811),
				Y: floatPtr(159.285),
				Z: nil,
				E: floatPtr(0.01274),
			},
		},
		{
			name:  "G1 with negative coordinates",
			input: "G1 X-10.5 Y-20.3 Z-1.2 E-0.5",
			expected: &GCodeCoordinates{
				X: floatPtr(-10.5),
				Y: floatPtr(-20.3),
				Z: floatPtr(-1.2),
				E: floatPtr(-0.5),
			},
		},
		{
			name:  "G1 with integer coordinates",
			input: "G1 X100 Y200 Z5 E1",
			expected: &GCodeCoordinates{
				X: floatPtr(100.0),
				Y: floatPtr(200.0),
				Z: floatPtr(5.0),
				E: floatPtr(1.0),
			},
		},
		{
			name:  "G1 with leading/trailing spaces",
			input: "  G1 X10.5 Y20.3  ",
			expected: &GCodeCoordinates{
				X: floatPtr(10.5),
				Y: floatPtr(20.3),
				Z: nil,
				E: nil,
			},
		},
		{
			name:  "G1 with comment",
			input: "G1 X24.811 Y159.285 E.01274 ; print move",
			expected: &GCodeCoordinates{
				X: floatPtr(24.811),
				Y: floatPtr(159.285),
				Z: nil,
				E: floatPtr(0.01274),
			},
		},
		{
			name:     "G1 with no coordinates",
			input:    "G1",
			expected: nil,
		},
		{
			name:     "Non-G1 command (G0)",
			input:    "G0 X10 Y20",
			expected: nil,
		},
		{
			name:     "Non-G1 command (M104)",
			input:    "M104 S200",
			expected: nil,
		},
		{
			name:     "Comment line",
			input:    "; This is a comment",
			expected: nil,
		},
		{
			name:     "Empty line",
			input:    "",
			expected: nil,
		},
		{
			name:     "Whitespace only",
			input:    "   ",
			expected: nil,
		},
		{
			name:     "G1 in middle of line (should not match)",
			input:    "M117 G1 test",
			expected: nil,
		},
		{
			name:  "G1 with very small decimal",
			input: "G1 X0.001 Y0.002 E0.00026",
			expected: &GCodeCoordinates{
				X: floatPtr(0.001),
				Y: floatPtr(0.002),
				Z: nil,
				E: floatPtr(0.00026),
			},
		},
		{
			name:  "G1 with zero values",
			input: "G1 X0 Y0 Z0 E0",
			expected: &GCodeCoordinates{
				X: floatPtr(0.0),
				Y: floatPtr(0.0),
				Z: floatPtr(0.0),
				E: floatPtr(0.0),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := p.parseGCodeLine(tt.input)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("Expected %+v, got nil", tt.expected)
				return
			}

			// Compare X coordinate
			if (tt.expected.X == nil) != (result.X == nil) {
				t.Errorf("X coordinate pointer mismatch: expected %v, got %v", tt.expected.X, result.X)
			} else if tt.expected.X != nil && *tt.expected.X != *result.X {
				t.Errorf("X coordinate value mismatch: expected %f, got %f", *tt.expected.X, *result.X)
			}

			// Compare Y coordinate
			if (tt.expected.Y == nil) != (result.Y == nil) {
				t.Errorf("Y coordinate pointer mismatch: expected %v, got %v", tt.expected.Y, result.Y)
			} else if tt.expected.Y != nil && *tt.expected.Y != *result.Y {
				t.Errorf("Y coordinate value mismatch: expected %f, got %f", *tt.expected.Y, *result.Y)
			}

			// Compare Z coordinate
			if (tt.expected.Z == nil) != (result.Z == nil) {
				t.Errorf("Z coordinate pointer mismatch: expected %v, got %v", tt.expected.Z, result.Z)
			} else if tt.expected.Z != nil && *tt.expected.Z != *result.Z {
				t.Errorf("Z coordinate value mismatch: expected %f, got %f", *tt.expected.Z, *result.Z)
			}

			// Compare E coordinate
			if (tt.expected.E == nil) != (result.E == nil) {
				t.Errorf("E coordinate pointer mismatch: expected %v, got %v", tt.expected.E, result.E)
			} else if tt.expected.E != nil && *tt.expected.E != *result.E {
				t.Errorf("E coordinate value mismatch: expected %f, got %f", *tt.expected.E, *result.E)
			}
		})
	}
}

func TestStreamingProcessor_findMarkerPositions_LastPrintCoordinates(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		gcodeContent string
		expectedX    float64
		expectedY    float64
		expectedZ    float64
		expectError  bool
	}{
		{
			name: "basic last print coordinates",
			gcodeContent: `G1 Z3.601
M211 X0 Y0 Z0 ;turn off soft endstop
M1007 S1
G1 X32.183 Y151.913 E.00026
G1 X24.811 Y159.285 E.01274
M625
G1 Z5.0`,
			expectedX: 24.811,
			expectedY: 159.285,
			expectedZ: 3.601,
		},
		{
			name: "a lot of command",
			gcodeContent: `G1 Z3.601
M211 X0 Y0 Z0 ;turn off soft endstop
M1007 S1
G1 X32.183 Y151.913 E.00026
G1 X24.811 Y159.285 E.01274
M625
G1 Z5.0
G1 Z4.601
M625
G1 X32.183 Y151.913 E.00026
G1 Z5.601
G1 Z6.601
M625
G1 X33.183 Y152.913
G1 X33.183 Y152.913 E-0.1
`,
			expectedX: 32.183,
			expectedY: 151.913,
			expectedZ: 4.601,
		},
		{
			name: "Z changes before print commands",
			gcodeContent: `M211 X0 Y0 Z0 ;turn off soft endstop
M1007 S1
G1 Z1.0
G1 Z2.0
G1 Z3.601
G1 X10.0 Y20.0 E0.1
G1 X24.811 Y159.285 E.01274
M625`,
			expectedX: 24.811,
			expectedY: 159.285,
			expectedZ: 3.601,
		},
		{
			name: "Z changes after last print command",
			gcodeContent: `G1 Z3.601
M211 X0 Y0 Z0 ;turn off soft endstop
M1007 S1
G1 X24.811 Y159.285 E.01274
G1 Z10.0
G1 Z20.0
M625`,
			expectedX: 24.811,
			expectedY: 159.285,
			expectedZ: 3.601,
		},
		{
			name: "multiple print commands with different Z",
			gcodeContent: `M211 X0 Y0 Z0 ;turn off soft endstop
M1007 S1
G1 Z1.0
G1 X10.0 Y20.0 E0.1
G1 Z2.0
G1 X15.0 Y25.0 E0.2
G1 Z3.601
G1 X24.811 Y159.285 E.01274
M625`,
			expectedX: 24.811,
			expectedY: 159.285,
			expectedZ: 3.601,
		},
		{
			name: "print command without X/Y coordinates",
			gcodeContent: `G1 Z3.601
M211 X0 Y0 Z0 ;turn off soft endstop
M1007 S1
G1 X24.811 Y159.285 E.01274
G1 E0.05
M625`,
			expectedX: 24.811,
			expectedY: 159.285,
			expectedZ: 3.601,
		},
		{
			name: "print command with only X coordinate",
			gcodeContent: `G1 Z3.601
M211 X0 Y0 Z0 ;turn off soft endstop
M1007 S1
G1 X10.0 Y20.0 E0.1
G1 X24.811 E.01274
M625`,
			expectedX: 24.811,
			expectedY: 20.0,
			expectedZ: 3.601,
		},
		{
			name: "print command with only Y coordinate",
			gcodeContent: `G1 Z3.601
M211 X0 Y0 Z0 ;turn off soft endstop
M1007 S1
G1 X10.0 Y20.0 E0.1
G1 Y159.285 E.01274
M625`,
			expectedX: 10.0,
			expectedY: 159.285,
			expectedZ: 3.601,
		},
		{
			name: "negative coordinates",
			gcodeContent: `G1 Z-1.5
M211 X0 Y0 Z0 ;turn off soft endstop
M1007 S1
G1 X-24.811 Y-159.285 E.01274
M625`,
			expectedX: -24.811,
			expectedY: -159.285,
			expectedZ: -1.5,
		},
		{
			name: "zero coordinates",
			gcodeContent: `G1 Z0
M211 X0 Y0 Z0 ;turn off soft endstop
M1007 S1
G1 X0 Y0 E0.01
M625`,
			expectedX: 0.0,
			expectedY: 0.0,
			expectedZ: 0.0,
		},
		{
			name: "no print commands (only non-extrusion moves)",
			gcodeContent: `G1 Z3.601
M211 X0 Y0 Z0 ;turn off soft endstop
M1007 S1
G1 X24.811 Y159.285
G1 X10.0 Y20.0 E0
G1 X5.0 Y10.0 E-0.1
M625`,
			expectedX: 0.0,
			expectedY: 0.0,
			expectedZ: 0.0,
		},
		{
			name: "Z coordinate in same line as print command",
			gcodeContent: `M211 X0 Y0 Z0 ;turn off soft endstop
M1007 S1
G1 X24.811 Y159.285 Z3.601 E.01274
M625`,
			expectedX: 24.811,
			expectedY: 159.285,
			expectedZ: 3.601,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create temporary file with test content
			tmpFile, err := os.CreateTemp(t.TempDir(), "gcode_test_*.gcode")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())
			defer tmpFile.Close()

			if _, err := tmpFile.WriteString(tt.gcodeContent); err != nil {
				t.Fatalf("Failed to write test content: %v", err)
			}
			tmpFile.Close()

			// Create processor with test configuration
			config := ProcessingRequest{
				Iterations: 1,
				Printer:    "unit-tests-gcode2",
			}
			processor, err := NewStreamingProcessor(config)
			if err != nil {
				t.Fatalf("Failed to create processor: %v", err)
			}

			// Test the function
			positions, err := processor.findMarkerPositions(tmpFile.Name())

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check coordinates
			if positions.LastPrintX != tt.expectedX {
				t.Errorf("LastPrintX: expected %f, got %f", tt.expectedX, positions.LastPrintX)
			}

			if positions.LastPrintY != tt.expectedY {
				t.Errorf("LastPrintY: expected %f, got %f", tt.expectedY, positions.LastPrintY)
			}

			if positions.LastPrintZ != tt.expectedZ {
				t.Errorf("LastPrintZ: expected %f, got %f", tt.expectedZ, positions.LastPrintZ)
			}
		})
	}
}
