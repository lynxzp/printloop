// file: processor_ProcessFile_test.go
package main

import (
	"reflect"
	"testing"
)

// Test core logic with simple string slices (no I/O)
func TestGCodeProcessor_ProcessLines(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ProcessingRequest{
				Iterations: 2,
			}
			processor := &GCodeProcessor{config: config}

			result, err := processor.ProcessLines(tt.input, tt.markers)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(result.Lines, tt.expected) {
				t.Errorf("Result mismatch.\nExpected:\n%v\n\nActual:\n%v",
					tt.expected, result.Lines)
			}
		})
	}
}
