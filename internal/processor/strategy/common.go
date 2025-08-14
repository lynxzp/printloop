package strategy

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type startMarkerMatch struct {
	begin int64
	end   int64
}

// FileScanner provides common file scanning functionality for strategies
type FileScanner struct {
	filePath string
	lines    []string
}

// NewFileScanner creates a new file scanner
func NewFileScanner(filePath string) (*FileScanner, error) {
	lines, err := readAllLines(filePath)
	if err != nil {
		return nil, err
	}
	return &FileScanner{
		filePath: filePath,
		lines:    lines,
	}, nil
}

// FindFirstMarkerFromStart finds the first occurrence of markers from the beginning
func (fs *FileScanner) FindFirstMarkerFromStart(markers []string) (int64, int64, error) {
	return fs.findMarkerWithSlidingWindow(markers, 0, len(fs.lines)-1, true)
}

// FindFirstMarkerFromLine finds the first occurrence of markers starting from a specific line
func (fs *FileScanner) FindFirstMarkerFromLine(markers []string, startLine int64) (int64, int64, error) {
	if startLine < 0 || int(startLine) >= len(fs.lines) {
		return 0, 0, fmt.Errorf("start line %d out of bounds", startLine)
	}
	return fs.findMarkerWithSlidingWindow(markers, int(startLine), len(fs.lines)-1, true)
}

// FindLastMarkerFromStart finds the last occurrence of markers from the beginning
func (fs *FileScanner) FindLastMarkerFromStart(markers []string) (int64, int64, error) {
	return fs.findMarkerWithSlidingWindow(markers, 0, len(fs.lines)-1, false)
}

// FindLastMarkerFromLine finds the last occurrence of markers starting from a specific line
func (fs *FileScanner) FindLastMarkerFromLine(markers []string, startLine int64) (int64, int64, error) {
	if startLine < 0 || int(startLine) >= len(fs.lines) {
		return 0, 0, fmt.Errorf("start line %d out of bounds", startLine)
	}
	return fs.findMarkerWithSlidingWindow(markers, int(startLine), len(fs.lines)-1, false)
}

// findMarkerWithSlidingWindow implements the sliding window search algorithm
func (fs *FileScanner) findMarkerWithSlidingWindow(markers []string, startIdx, endIdx int, returnFirst bool) (int64, int64, error) {
	if len(markers) == 0 {
		return 0, 0, fmt.Errorf("no markers provided")
	}

	var lastFoundBegin, lastFoundEnd int64 = -1, -1

	if len(markers) == 1 {
		// Single line marker - simple search
		for i := startIdx; i <= endIdx; i++ {
			if strings.Contains(strings.TrimSpace(fs.lines[i]), strings.TrimSpace(markers[0])) {
				if returnFirst {
					return int64(i), int64(i), nil
				}
				lastFoundBegin = int64(i)
				lastFoundEnd = int64(i)
			}
		}
	} else {
		// Multiline marker - sliding window approach
		window := make([]string, 0, len(markers)+10)
		
		for i := startIdx; i <= endIdx; i++ {
			line := fs.lines[i]
			window = append(window, line)

			// Keep window size reasonable
			maxWindowSize := len(markers) + 10
			if len(window) > maxWindowSize {
				window = window[1:] // Remove oldest line
			}

			// Calculate the correct window start line position
			currentWindowStart := int64(i) - int64(len(window)) + 1

			// Try to find marker pattern in current window
			if matchPos := findStartMarkerInWindow(window, markers, currentWindowStart); matchPos != nil {
				if returnFirst {
					return matchPos.begin, matchPos.end, nil
				}
				lastFoundBegin = matchPos.begin
				lastFoundEnd = matchPos.end
			}
		}
	}

	if lastFoundBegin == -1 {
		return 0, 0, fmt.Errorf("marker not found: %v", markers)
	}

	return lastFoundBegin, lastFoundEnd, nil
}

// readAllLines reads all lines from a file into memory
func readAllLines(filePath string) ([]string, error) {
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

// findStartMarkerInWindow searches for start marker pattern in the sliding window
func findStartMarkerInWindow(window []string, markers []string, windowStartLine int64) *startMarkerMatch {
	if len(markers) == 1 {
		// Single line marker
		for i, line := range window {
			if strings.Contains(strings.TrimSpace(line), strings.TrimSpace(markers[0])) {
				pos := windowStartLine + int64(i)
				return &startMarkerMatch{begin: pos, end: pos}
			}
		}
		return nil
	}

	// Multiline marker search
	for startIdx := range window {
		if match := tryMatchMultilineStart(window, startIdx, windowStartLine, markers); match != nil {
			return match
		}
	}
	return nil
}

// tryMatchMultilineStart attempts to match multiline start marker from given position
func tryMatchMultilineStart(window []string, startIdx int, windowStartLine int64, markers []string) *startMarkerMatch {
	windowIdx := startIdx
	markerIdx := 0
	firstMarkerLine := int64(-1)
	lastMarkerLine := int64(-1)

	for markerIdx < len(markers) && windowIdx < len(window) {
		cleanLine := strings.TrimSpace(window[windowIdx])
		cleanMarker := strings.TrimSpace(markers[markerIdx])

		if strings.Contains(cleanLine, cleanMarker) {
			currentLine := windowStartLine + int64(windowIdx)
			if firstMarkerLine == -1 {
				firstMarkerLine = currentLine
			}
			lastMarkerLine = currentLine
			markerIdx++
			windowIdx++
		} else if cleanLine == "" || strings.HasPrefix(cleanLine, ";") {
			// Skip empty or comment lines
			windowIdx++
		} else {
			// This line doesn't match and isn't skippable
			return nil
		}
	}

	if markerIdx == len(markers) {
		return &startMarkerMatch{begin: firstMarkerLine, end: lastMarkerLine}
	}
	return nil
}
