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
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	// Read all lines into memory for easier processing
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return 0, 0, err
	}

	lastFoundBegin := int64(-1)
	lastFoundEnd := int64(-1)

	if len(markers) == 1 {
		// Single line marker - find last occurrence
		marker := strings.TrimSpace(markers[0])
		for i, line := range lines {
			if strings.Contains(strings.TrimSpace(line), marker) {
				lastFoundBegin = int64(i)
				lastFoundEnd = int64(i)
			}
		}
	} else {
		// Multiline marker - scan from each position and try to match the pattern
		for startPos := 0; startPos <= len(lines)-len(markers); startPos++ {
			if match := s.tryMatchMultilinePattern(lines, startPos, markers); match != nil {
				lastFoundBegin = match.begin
				lastFoundEnd = match.end
			}
		}
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

	// Read all lines into memory for easier processing
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return 0, 0, err
	}

	lastFoundBegin := int64(-1)
	lastFoundEnd := int64(-1)

	if len(markers) == 1 {
		// Single line marker - find last occurrence after searchFromLine
		marker := strings.TrimSpace(markers[0])
		for i := int(searchFromLine) + 1; i < len(lines); i++ {
			if strings.Contains(strings.TrimSpace(lines[i]), marker) {
				lastFoundBegin = int64(i)
				lastFoundEnd = int64(i)
			}
		}
	} else {
		// Multiline marker - scan from searchFromLine+1 and try to match the pattern
		for startPos := int(searchFromLine) + 1; startPos <= len(lines)-len(markers); startPos++ {
			if match := s.tryMatchMultilinePattern(lines, startPos, markers); match != nil {
				lastFoundBegin = match.begin
				lastFoundEnd = match.end
			}
		}
	}

	if lastFoundBegin == -1 {
		return 0, 0, fmt.Errorf("end marker not found after line %d: %v", searchFromLine, markers)
	}

	return lastFoundBegin, lastFoundEnd, nil
}

// tryMatchMultilinePattern attempts to match multiline pattern starting from given position
func (s *AfterLastAppearStrategy) tryMatchMultilinePattern(lines []string, startPos int, markers []string) *startMarkerMatch {
	linePos := startPos
	markerIdx := 0

	for markerIdx < len(markers) && linePos < len(lines) {
		cleanLine := strings.TrimSpace(lines[linePos])
		cleanMarker := strings.TrimSpace(markers[markerIdx])

		if strings.Contains(cleanLine, cleanMarker) {
			markerIdx++
			linePos++
		} else if cleanLine == "" || strings.HasPrefix(cleanLine, ";") {
			// Skip empty or comment lines
			linePos++
		} else {
			// This line doesn't match and isn't skippable
			return nil
		}
	}

	if markerIdx == len(markers) {
		return &startMarkerMatch{
			begin: int64(startPos),
			end:   int64(linePos - 1),
		}
	}
	return nil
}
