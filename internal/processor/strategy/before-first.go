package strategy

import (
	"bufio"
	"fmt"
	"os"
)

// BeforeCommandStrategy finds markers that appear before specific commands
type BeforeCommandStrategy struct{}

func (s *BeforeCommandStrategy) FindInitSectionPosition(filePath string, markers []string) (int64, int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := int64(0)

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
			return matchPos.begin, matchPos.end, nil
		}

		lineNum++
	}

	return 0, 0, fmt.Errorf("start marker not found before commands: %v", markers)
}

func (s *BeforeCommandStrategy) FindPrintSectionPosition(filePath string, markers []string, searchFromLine int64) (int64, int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := int64(0)

	// Skip to the search start position
	for lineNum <= searchFromLine && scanner.Scan() {
		lineNum++
	}

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

		// Calculate the correct window start line position for this iteration
		currentWindowStart := lineNum - int64(len(window)) + 1

		// Try to find marker pattern in current window
		if matchPos := findStartMarkerInWindow(window, markers, currentWindowStart); matchPos != nil {
			return matchPos.begin, matchPos.end, nil
		}

		lineNum++
	}

	return 0, 0, fmt.Errorf("end marker not found before commands after line %d: %v", searchFromLine, markers)
}
