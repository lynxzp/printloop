// file: internal/processor/strategy/after-last.go
package strategy

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// AfterLastAppearStrategy finds the last appearance of markers
type AfterLastAppearStrategy struct{}

func (s *AfterLastAppearStrategy) FindInitSectionPosition(filePath string, markers []string) (int64, int64, error) {
	// For init section, last appear means find the last occurrence of the complete pattern
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := int64(0)
	lastFoundBegin := int64(-1)
	lastFoundEnd := int64(-1)

	// Sliding window for multiline marker detection
	window := make([]string, 0, len(markers)+10)

	for scanner.Scan() {
		line := scanner.Text()
		window = append(window, line)

		// Keep window size reasonable
		maxWindowSize := len(markers) + 10
		if len(window) > maxWindowSize {
			window = window[1:] // Remove oldest line
		}

		// Try to find start marker pattern in current window
		if matchPos := findStartMarkerInWindow(window, markers, lineNum-int64(len(window))+1); matchPos != nil {
			lastFoundBegin = matchPos.begin
			lastFoundEnd = matchPos.end
		}

		lineNum++
	}

	if lastFoundBegin == -1 {
		return 0, 0, fmt.Errorf("start marker not found: %v", markers)
	}

	return lastFoundBegin, lastFoundEnd, nil
}

func (s *AfterLastAppearStrategy) FindPrintSectionPosition(filePath string, markers []string, searchFromLine int64) (int64, int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := int64(0)
	lastFoundBegin := int64(-1)
	lastFoundEnd := int64(-1)

	// Skip to the search start position
	for lineNum <= searchFromLine && scanner.Scan() {
		lineNum++
	}

	// For single line markers, use simple approach
	if len(markers) == 1 {
		marker := strings.TrimSpace(markers[0])
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(strings.TrimSpace(line), marker) {
				lastFoundBegin = lineNum
				lastFoundEnd = lineNum
			}
			lineNum++
		}
	} else {
		// For multiline markers, use sliding window
		window := make([]string, 0, len(markers)+10)

		for scanner.Scan() {
			line := scanner.Text()
			window = append(window, line)

			// Keep window size reasonable
			maxWindowSize := len(markers) + 10
			if len(window) > maxWindowSize {
				window = window[1:] // Remove oldest line
			}

			// Calculate correct window start position
			windowStart := lineNum - int64(len(window)) + 1

			// Try to find marker pattern in current window
			if matchPos := findStartMarkerInWindow(window, markers, windowStart); matchPos != nil {
				lastFoundBegin = matchPos.begin
				lastFoundEnd = matchPos.end
			}

			lineNum++
		}
	}

	if lastFoundBegin == -1 {
		return 0, 0, fmt.Errorf("end marker not found after line %d: %v", searchFromLine, markers)
	}

	return lastFoundBegin, lastFoundEnd, nil
}
