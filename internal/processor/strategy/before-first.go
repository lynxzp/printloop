package strategy

import (
	"bufio"
	"fmt"
	"os"
	"strings"
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

func (s *BeforeCommandStrategy) FindPrintSectionPosition(filePath string, marker string, searchFromLine int64) (int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := int64(0)

	// Skip to the search start position
	for lineNum <= searchFromLine && scanner.Scan() {
		lineNum++
	}

	// Find first occurrence after searchFromLine
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(strings.TrimSpace(line), strings.TrimSpace(marker)) {
			return lineNum, nil
		}
		lineNum++
	}

	return 0, fmt.Errorf("end marker '%s' not found before commands after line %d", marker, searchFromLine)
}

// findStartMarkerInWindowReverse searches for start marker pattern in reverse order
func findStartMarkerInWindowReverse(window []string, markers []string, windowStartLine int64) *startMarkerMatch {
	if len(markers) == 1 {
		// Single line marker - search from end of window backwards
		for i := len(window) - 1; i >= 0; i-- {
			line := window[i]
			if strings.Contains(strings.TrimSpace(line), strings.TrimSpace(markers[0])) {
				pos := windowStartLine + int64(i)
				return &startMarkerMatch{begin: pos, end: pos}
			}
		}
		return nil
	}

	// Multiline marker search in reverse
	for startIdx := len(window) - 1; startIdx >= 0; startIdx-- {
		if match := tryMatchMultilineStartReverse(window, startIdx, windowStartLine, markers); match != nil {
			return match
		}
	}
	return nil
}

// tryMatchMultilineStartReverse attempts to match multiline start marker in reverse from given position
func tryMatchMultilineStartReverse(window []string, startIdx int, windowStartLine int64, markers []string) *startMarkerMatch {
	windowIdx := startIdx
	markerIdx := len(markers) - 1 // Start from last marker and go backwards
	firstMarkerLine := int64(-1)
	lastMarkerLine := int64(-1)

	for markerIdx >= 0 && windowIdx >= 0 {
		cleanLine := strings.TrimSpace(window[windowIdx])
		cleanMarker := strings.TrimSpace(markers[markerIdx])

		if strings.Contains(cleanLine, cleanMarker) {
			currentLine := windowStartLine + int64(windowIdx)
			if lastMarkerLine == -1 {
				lastMarkerLine = currentLine
			}
			firstMarkerLine = currentLine
			markerIdx--
			windowIdx--
		} else if cleanLine == "" || strings.HasPrefix(cleanLine, ";") {
			// Skip empty or comment lines
			windowIdx--
		} else {
			// This line doesn't match and isn't skippable
			return nil
		}
	}

	if markerIdx < 0 {
		return &startMarkerMatch{begin: firstMarkerLine, end: lastMarkerLine}
	}
	return nil
}
