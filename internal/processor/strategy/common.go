package strategy

import "strings"

type startMarkerMatch struct {
	begin int64
	end   int64
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
	for startIdx := 0; startIdx < len(window); startIdx++ {
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
