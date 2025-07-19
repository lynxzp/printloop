package strategy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAfterFirstAppearStrategy(t *testing.T) {
	tests := []struct {
		name               string
		fileContent        []string
		initMarkers        []string
		printMarkers       []string
		searchFromLine     int64
		expectedInitFirst  int64
		expectedInitLast   int64
		expectedPrintFirst int64
		expectedPrintLast  int64
		expectInitError    bool
		expectPrintError   bool
	}{
		{
			name: "single line markers - single occurrence",
			fileContent: []string{
				"HEADER",
				"START_PRINT",
				"BODY",
				"END_PRINT",
				"FOOTER",
			},
			initMarkers:        []string{"START_PRINT"},
			printMarkers:       []string{"END_PRINT"},
			searchFromLine:     1,
			expectedInitFirst:  1,
			expectedInitLast:   1,
			expectedPrintFirst: 3,
			expectedPrintLast:  3,
		},
		{
			name: "single line markers - multiple occurrences",
			fileContent: []string{
				"HEADER",
				"START_PRINT",
				"BODY1",
				"END_PRINT",
				"BODY2",
				"END_PRINT",
				"FOOTER",
			},
			initMarkers:        []string{"START_PRINT"},
			printMarkers:       []string{"END_PRINT"},
			searchFromLine:     1,
			expectedInitFirst:  1,
			expectedInitLast:   1,
			expectedPrintFirst: 3, // Should find FIRST occurrence
			expectedPrintLast:  3,
		},
		{
			name: "multiline init markers - single occurrence",
			fileContent: []string{
				"HEADER",
				"START_LINE1",
				"START_LINE2",
				"BODY",
				"END_PRINT",
				"FOOTER",
			},
			initMarkers:        []string{"START_LINE1", "START_LINE2"},
			printMarkers:       []string{"END_PRINT"},
			searchFromLine:     2,
			expectedInitFirst:  1,
			expectedInitLast:   2,
			expectedPrintFirst: 4,
			expectedPrintLast:  4,
		},
		{
			name: "multiline init markers - multiple occurrences",
			fileContent: []string{
				"HEADER",
				"START_LINE1",
				"START_LINE2",
				"BODY1",
				"START_LINE1",
				"START_LINE2",
				"BODY2",
				"END_PRINT",
				"FOOTER",
			},
			initMarkers:        []string{"START_LINE1", "START_LINE2"},
			printMarkers:       []string{"END_PRINT"},
			searchFromLine:     2,
			expectedInitFirst:  1, // Should find FIRST occurrence
			expectedInitLast:   2,
			expectedPrintFirst: 7,
			expectedPrintLast:  7,
		},
		{
			name: "multiline print markers - multiple occurrences",
			fileContent: []string{
				"HEADER",
				"START_PRINT",
				"BODY1",
				"END_LINE1",
				"END_LINE2",
				"BODY2",
				"END_LINE1",
				"END_LINE2",
				"FOOTER",
			},
			initMarkers:        []string{"START_PRINT"},
			printMarkers:       []string{"END_LINE1", "END_LINE2"},
			searchFromLine:     1,
			expectedInitFirst:  1,
			expectedInitLast:   1,
			expectedPrintFirst: 3, // Should find FIRST occurrence
			expectedPrintLast:  4,
		},
		{
			name: "markers with spaces and comments",
			fileContent: []string{
				"HEADER",
				" START_PRINT ",
				"BODY",
				"END_PRINT ; comment",
				"BODY2",
				" END_PRINT ; another comment",
				"FOOTER",
			},
			initMarkers:        []string{"START_PRINT"},
			printMarkers:       []string{"END_PRINT"},
			searchFromLine:     1,
			expectedInitFirst:  1,
			expectedInitLast:   1,
			expectedPrintFirst: 3, // Should find FIRST occurrence
			expectedPrintLast:  3,
		},
		{
			name: "multiline with empty lines and comments",
			fileContent: []string{
				"HEADER",
				"START_LINE1",
				"; comment",
				" ",
				"START_LINE2",
				"BODY1",
				"START_LINE1",
				"",
				"; another comment",
				"START_LINE2",
				"BODY2",
				"END_PRINT",
				"FOOTER",
			},
			initMarkers:        []string{"START_LINE1", "START_LINE2"},
			printMarkers:       []string{"END_PRINT"},
			searchFromLine:     4,
			expectedInitFirst:  1, // Should find FIRST occurrence
			expectedInitLast:   4,
			expectedPrintFirst: 11,
			expectedPrintLast:  11,
		},
		{
			name: "init marker not found",
			fileContent: []string{
				"HEADER",
				"BODY",
				"END_PRINT",
				"FOOTER",
			},
			initMarkers:     []string{"START_PRINT"},
			printMarkers:    []string{"END_PRINT"},
			searchFromLine:  0,
			expectInitError: true,
		},
		{
			name: "print marker not found after search line",
			fileContent: []string{
				"HEADER",
				"START_PRINT",
				"END_PRINT", // Before search line
				"BODY",
				"FOOTER",
			},
			initMarkers:       []string{"START_PRINT"},
			printMarkers:      []string{"END_PRINT"},
			searchFromLine:    3, // Search after END_PRINT
			expectedInitFirst: 1,
			expectedInitLast:  1,
			expectPrintError:  true,
		},
		{
			name: "partial multiline match - should not find",
			fileContent: []string{
				"HEADER",
				"START_LINE1",
				"WRONG_LINE",
				"START_LINE2",
				"BODY",
				"END_PRINT",
				"FOOTER",
			},
			initMarkers:     []string{"START_LINE1", "START_LINE2"},
			printMarkers:    []string{"END_PRINT"},
			searchFromLine:  3,
			expectInitError: true,
		},
		{
			name: "multiple init markers - find first complete match",
			fileContent: []string{
				"HEADER",
				"START_LINE1", // First incomplete match
				"WRONG_LINE",
				"START_LINE1", // Second complete match
				"START_LINE2",
				"BODY",
				"END_PRINT",
				"FOOTER",
			},
			initMarkers:        []string{"START_LINE1", "START_LINE2"},
			printMarkers:       []string{"END_PRINT"},
			searchFromLine:     4,
			expectedInitFirst:  3, // Should find first complete match
			expectedInitLast:   4,
			expectedPrintFirst: 6,
			expectedPrintLast:  6,
		},
		{
			name: "three occurrences - find first",
			fileContent: []string{
				"HEADER",
				"START_PRINT",
				"BODY1",
				"END_PRINT", // First - should find this
				"BODY2",
				"END_PRINT", // Second
				"BODY3",
				"END_PRINT", // Third
				"FOOTER",
			},
			initMarkers:        []string{"START_PRINT"},
			printMarkers:       []string{"END_PRINT"},
			searchFromLine:     1,
			expectedInitFirst:  1,
			expectedInitLast:   1,
			expectedPrintFirst: 3, // Should find FIRST occurrence
			expectedPrintLast:  3,
		},
		{
			name: "search from line limits results",
			fileContent: []string{
				"HEADER",
				"START_PRINT",
				"BODY1",
				"END_PRINT", // Before search line
				"BODY2",
				"END_PRINT", // After search line - should find this
				"BODY3",
				"END_PRINT", // Also after search line
				"FOOTER",
			},
			initMarkers:        []string{"START_PRINT"},
			printMarkers:       []string{"END_PRINT"},
			searchFromLine:     4, // Search after first END_PRINT
			expectedInitFirst:  1,
			expectedInitLast:   1,
			expectedPrintFirst: 5, // Should find first after search line
			expectedPrintLast:  5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "test.txt")

			file, err := os.Create(testFile)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			for _, line := range tt.fileContent {
				if _, err := file.WriteString(line + "\n"); err != nil {
					t.Fatalf("Failed to write test content: %v", err)
				}
			}
			file.Close()

			strategy := &AfterFirstAppearStrategy{}

			// Test FindInitSectionPosition
			initFirst, initLast, initErr := strategy.FindInitSectionPosition(testFile, tt.initMarkers)

			if tt.expectInitError {
				if initErr == nil {
					t.Errorf("Expected init error but got none")
				}
			} else {
				if initErr != nil {
					t.Errorf("Unexpected init error: %v", initErr)
				} else {
					if initFirst != tt.expectedInitFirst {
						t.Errorf("Init first line: expected %d, got %d", tt.expectedInitFirst, initFirst)
					}
					if initLast != tt.expectedInitLast {
						t.Errorf("Init last line: expected %d, got %d", tt.expectedInitLast, initLast)
					}
				}
			}

			// Test FindPrintSectionPosition
			if !tt.expectInitError && !tt.expectPrintError {
				printFirst, printLast, printErr := strategy.FindPrintSectionPosition(testFile, tt.printMarkers, tt.searchFromLine)

				if printErr != nil {
					t.Errorf("Unexpected print error: %v", printErr)
				} else {
					if printFirst != tt.expectedPrintFirst {
						t.Errorf("Print first line: expected %d, got %d", tt.expectedPrintFirst, printFirst)
					}
					if printLast != tt.expectedPrintLast {
						t.Errorf("Print last line: expected %d, got %d", tt.expectedPrintLast, printLast)
					}
				}
			} else if tt.expectPrintError {
				_, _, printErr := strategy.FindPrintSectionPosition(testFile, tt.printMarkers, tt.searchFromLine)
				if printErr == nil {
					t.Errorf("Expected print error but got none")
				}
			}
		})
	}
}
