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
			name: "sample 1",
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
				StartMarker: "START_PRINT",
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
			name: "sample 2",
			input: []string{
				"HEADER",
				"START_PRINT",
				"BODY",
				"END_PRINT",
				"FOOTER",
			},
			markers: PositionMarkers{
				StartMarker: "START_PRINT",
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
			name: "multi end",
			input: []string{
				"HEADER",
				"START_PRINT",
				"BODY",
				"END_PRINT",
				"BODY",
				"END_PRINT",
				"FOOTER",
			},
			markers: PositionMarkers{
				StartMarker: "START_PRINT",
				EndMarker:   "END_PRINT",
			},
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
			name:  "missing start marker",
			input: []string{"HEADER", "BODY", "END_PRINT"},
			markers: PositionMarkers{
				StartMarker: "START_PRINT",
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
