package main

//
//import (
//	"bufio"
//	"io"
//	"strings"
//	"testing"
//)
//
//func TestFindGCodePositions(t *testing.T) {
//	tests := []struct {
//		name          string
//		input         string
//		expectError   bool
//		errorContains string
//	}{
//		{
//			name: "valid sequence with both positions",
//			input: `G28 ; home all axes
//M211 X0 Y0 Z0 ;turn off soft endstop
//M1007 S1
//G90
//G1 X10 Y10
//G625 S100
//G1 X20 Y20`,
//			expectError: false,
//		},
//		{
//			name: "sequence with comments and empty lines between",
//			input: `M211 X0 Y0 Z0 ;turn off soft endstop
//; this is a comment
//
//M1007 S1
//; another comment
//
//G90
//G625 S200`,
//			expectError: false,
//		},
//		{
//			name: "multiple G625 lines (should return last one)",
//			input: `M211 X0 Y0 Z0 ;turn off soft endstop
//M1007 S1
//G90
//G625 S100
//G1 X10 Y10
//G625 S200
//G1 X20 Y20
//G625 S300`,
//			expectError: false,
//		},
//		{
//			name: "missing M211 line",
//			input: `G28 ; home all axes
//M1007 S1
//G90
//G625 S100`,
//			expectError:   true,
//			errorContains: "failed to find required G-code positions",
//		},
//		{
//			name: "missing M1007 line",
//			input: `M211 X0 Y0 Z0 ;turn off soft endstop
//G90
//G625 S100`,
//			expectError:   true,
//			errorContains: "failed to find required G-code positions",
//		},
//		{
//			name: "missing G90 line",
//			input: `M211 X0 Y0 Z0 ;turn off soft endstop
//M1007 S1
//G1 X10 Y10
//G625 S100`,
//			expectError:   true,
//			errorContains: "failed to find required G-code positions",
//		},
//		{
//			name: "missing G625 line",
//			input: `M211 X0 Y0 Z0 ;turn off soft endstop
//M1007 S1
//G90
//G1 X10 Y10`,
//			expectError:   true,
//			errorContains: "failed to find second G-code position (G625)",
//		},
//		{
//			name: "wrong order - M1007 before M211",
//			input: `M1007 S1
//M211 X0 Y0 Z0 ;turn off soft endstop
//G90
//G625 S100`,
//			expectError:   true,
//			errorContains: "failed to find required G-code positions",
//		},
//		{
//			name: "interrupted sequence - other command between M1007 and G90",
//			input: `M211 X0 Y0 Z0 ;turn off soft endstop
//M1007 S1
//G1 X10 Y10
//G90
//G625 S100`,
//			expectError:   true,
//			errorContains: "failed to find required G-code positions",
//		},
//		{
//			name: "interrupted sequence - other command between M211 and M1007",
//			input: `M211 X0 Y0 Z0 ;turn off soft endstop
//G1 X10 Y10
//M1007 S1
//G90
//G625 S100`,
//			expectError:   true,
//			errorContains: "failed to find required G-code positions",
//		},
//		{
//			name: "sequence reset and then valid sequence",
//			input: `M211 X0 Y0 Z0 ;turn off soft endstop
//G1 X10 Y10
//M211 X0 Y0 Z0 ;turn off soft endstop
//M1007 S1
//G90
//G625 S100`,
//			expectError: false,
//		},
//		{
//			name:          "empty file",
//			input:         ``,
//			expectError:   true,
//			errorContains: "failed to find required G-code positions",
//		},
//		{
//			name: "only comments",
//			input: `; comment 1
//; comment 2
//; comment 3`,
//			expectError:   true,
//			errorContains: "failed to find required G-code positions",
//		},
//		{
//			name: "G625 in comment (should not match)",
//			input: `M211 X0 Y0 Z0 ;turn off soft endstop
//M1007 S1
//G90
//; G625 S100 this is in comment
//G1 X10 Y10`,
//			expectError:   true,
//			errorContains: "failed to find second G-code position (G625)",
//		},
//		{
//			name: "G625 mixed with comments",
//			input: `M211 X0 Y0 Z0 ;turn off soft endstop
//M1007 S1
//G90
//; G625 S100 this is in comment
//G625 S200
//; another comment with G625
//G1 X10 Y10`,
//			expectError: false,
//		},
//		{
//			name: "case sensitivity test",
//			input: `m211 x0 y0 z0 ;turn off soft endstop
//m1007 s1
//g90
//g625 s100`,
//			expectError:   true,
//			errorContains: "failed to find required G-code positions",
//		},
//		{
//			name: "extra whitespace in lines",
//			input: `  M211 X0 Y0 Z0 ;turn off soft endstop
//  M1007 S1
//  G90
//  G625 S100  `,
//			expectError: false,
//		},
//		{
//			name: "G625 appears before and after sequence",
//			input: `G625 S50
//M211 X0 Y0 Z0 ;turn off soft endstop
//M1007 S1
//G90
//G625 S100`,
//			expectError: false,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			reader := strings.NewReader(tt.input)
//
//			// Calculate expected positions using helper function
//			expectedFirst, expectedSecond := calculatePositions(tt.input)
//
//			first, second, err := FindGCodePositions(reader)
//
//			if tt.expectError {
//				if err == nil {
//					t.Errorf("expected error but got none")
//					return
//				}
//				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
//					t.Errorf("expected error containing '%s', got '%s'", tt.errorContains, err.Error())
//				}
//				return
//			}
//
//			if err != nil {
//				t.Errorf("unexpected error: %v", err)
//				return
//			}
//
//			if first != expectedFirst {
//				t.Errorf("expected first position %d, got %d", expectedFirst, first)
//			}
//
//			if second != expectedSecond {
//				t.Errorf("expected second position %d, got %d", expectedSecond, second)
//			}
//
//			// Verify that we can actually seek to these positions and read valid data
//			if first > 0 {
//				_, err := reader.Seek(first, io.SeekStart)
//				if err != nil {
//					t.Errorf("failed to seek to first position: %v", err)
//				}
//			}
//
//			if second > 0 {
//				_, err := reader.Seek(second, io.SeekStart)
//				if err != nil {
//					t.Errorf("failed to seek to second position: %v", err)
//				}
//			}
//		})
//	}
//}
//
//// Helper function to calculate expected positions by simulating the exact same logic
//func calculatePositions(input string) (int64, int64) {
//	reader := strings.NewReader(input)
//	scanner := bufio.NewScanner(reader)
//	var currentPos int64 = 0
//
//	// State machine identical to main function
//	const (
//		StateStart = iota
//		StateFoundM211
//		StateFoundM1007
//	)
//
//	state := StateStart
//	var posAfterM1007 int64 = -1
//	var firstFound bool
//	var first int64
//	var second int64
//
//	for scanner.Scan() {
//		line := scanner.Text()
//		lineBytes := scanner.Bytes()
//		lineLength := int64(len(lineBytes))
//
//		// Add 1 for newline character (scanner removes it)
//		nextPos := currentPos + lineLength + 1
//
//		// Clean the line (remove leading/trailing whitespace)
//		cleanLine := strings.TrimSpace(line)
//
//		// Skip empty lines and comment-only lines for sequence matching
//		isEmptyOrComment := cleanLine == "" || strings.HasPrefix(cleanLine, ";")
//
//		// Process line based on current state for first position
//		if !firstFound {
//			switch state {
//			case StateStart:
//				if strings.Contains(cleanLine, "M211 X0 Y0 Z0") && strings.Contains(cleanLine, "turn off soft endstop") {
//					state = StateFoundM211
//				}
//
//			case StateFoundM211:
//				if isEmptyOrComment {
//					// Skip empty lines and comments
//				} else if strings.TrimSpace(cleanLine) == "M1007 S1" {
//					state = StateFoundM1007
//					posAfterM1007 = nextPos
//				} else {
//					// Reset if we find something else
//					state = StateStart
//				}
//
//			case StateFoundM1007:
//				if isEmptyOrComment {
//					// Skip empty lines and comments
//				} else if strings.TrimSpace(cleanLine) == "G90" {
//					// Found complete sequence!
//					first = posAfterM1007
//					firstFound = true
//				} else {
//					// Reset if we find something else
//					state = StateStart
//				}
//			}
//		}
//
//		// Look for G625 lines for second position (skip comments)
//		if !isEmptyOrComment && strings.Contains(cleanLine, "G625") {
//			second = nextPos
//		}
//
//		currentPos = nextPos
//	}
//
//	return first, second
//}
