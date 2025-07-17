package processor

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"printloop/internal/types"
	"regexp"
	"strconv"
	"strings"
)

type PositionMarkers struct {
	StartMarker []string // For multiline markers, each element is a line
	EndMarker   string   // Keep single line for now
}

type StreamingProcessor struct {
	config  types.ProcessingRequest
	markers PositionMarkers
}

// MarkerPositions represents the found positions of start and end markers
type MarkerPositions struct {
	StartMarkerBegin int64   // First line of start marker (0-based)
	StartMarkerEnd   int64   // Last line of start marker (0-based)
	EndMarkerPos     int64   // Position of end marker (0-based)
	LastPrintX       float64 // X coordinate from last print command (G1 with positive E)
	LastPrintY       float64 // Y coordinate from last print command (G1 with positive E)
	LastPrintZ       float64 // Z coordinate that was active during last print command
}

// GCodeCoordinates holds parsed G-code coordinates
type GCodeCoordinates struct {
	X *float64
	Y *float64
	Z *float64
	E *float64
}

func NewStreamingProcessor(config types.ProcessingRequest, markers PositionMarkers) *StreamingProcessor {
	return &StreamingProcessor{
		config:  config,
		markers: markers,
	}
}

// ProcessFile processes a file using true streaming with multiple passes
func (p *StreamingProcessor) ProcessFile(inputPath, outputPath string) error {
	// Validate input first
	if err := p.validateInput(); err != nil {
		return err
	}

	// Pass 1: Find marker positions and extract G-code coordinates
	positions, err := p.findMarkerPositions(inputPath)
	if err != nil {
		return err
	}

	// Open output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	writer := bufio.NewWriter(outputFile)
	defer writer.Flush()

	// Pass 2: Stream header (lines 0 to StartMarkerEnd inclusive)
	if err := p.streamLinesRange(inputPath, writer, 0, positions.StartMarkerEnd, true); err != nil {
		return fmt.Errorf("failed to stream header: %w", err)
	}

	// Pass 3: For each iteration, stream body + end marker + generated content
	for i := int64(0); i < p.config.Iterations; i++ {
		// Stream body (lines after StartMarkerEnd to before EndMarkerPos)
		if positions.StartMarkerEnd+1 < positions.EndMarkerPos {
			if err := p.streamLinesRange(inputPath, writer, positions.StartMarkerEnd+1, positions.EndMarkerPos-1, false); err != nil {
				return fmt.Errorf("failed to stream body for iteration %d: %w", i+1, err)
			}
		}

		// Stream end marker line
		if err := p.streamLinesRange(inputPath, writer, positions.EndMarkerPos, positions.EndMarkerPos, false); err != nil {
			return fmt.Errorf("failed to stream end marker for iteration %d: %w", i+1, err)
		}

		// Stream generated content
		if err := p.streamGeneratedContent(writer, i+1); err != nil {
			return fmt.Errorf("failed to stream generated content for iteration %d: %w", i+1, err)
		}
	}

	// Pass 4: Stream footer (lines after EndMarkerPos to EOF)
	if err := p.streamLinesFromPosition(inputPath, writer, positions.EndMarkerPos+1); err != nil {
		return fmt.Errorf("failed to stream footer: %w", err)
	}

	return nil
}

// findMarkerPositions uses multiple passes to find marker positions and extract G-code coordinates
func (p *StreamingProcessor) findMarkerPositions(filePath string) (*MarkerPositions, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := int64(0)

	// Initialize result
	positions := &MarkerPositions{
		StartMarkerBegin: -1,
		StartMarkerEnd:   -1,
		EndMarkerPos:     -1,
	}

	// Tracking variables for G-code coordinates
	var lastPrintX, lastPrintY, lastPrintZ *float64
	var currentZ *float64 // Track current Z coordinate as we scan

	// Sliding window for start marker detection
	window := make([]string, 0, len(p.markers.StartMarker)+10)
	startMarkerFound := false

	for scanner.Scan() {
		line := scanner.Text()

		// Parse G-code coordinates from this line
		if coords := p.parseGCodeLine(line); coords != nil {
			// Update current Z from any G1 command
			if coords.Z != nil {
				currentZ = coords.Z
			}

			// Update last print coordinates (G1 with positive E)
			if coords.E != nil && *coords.E > 0 {
				// This is a print command - update print coordinates
				if coords.X != nil {
					lastPrintX = coords.X
				}
				if coords.Y != nil {
					lastPrintY = coords.Y
				}
				// Remember the Z that was active during this print command
				if currentZ != nil {
					lastPrintZ = currentZ
				}
			}
		}

		// Start marker detection using sliding window
		if !startMarkerFound {
			window = append(window, line)

			// Keep window size reasonable
			maxWindowSize := len(p.markers.StartMarker) + 10
			if len(window) > maxWindowSize {
				window = window[1:] // Remove oldest line
			}

			// Try to find start marker pattern in current window
			if matchPos := p.findStartMarkerInWindow(window, lineNum-int64(len(window))+1); matchPos != nil {
				positions.StartMarkerBegin = matchPos.begin
				positions.StartMarkerEnd = matchPos.end
				startMarkerFound = true
			}
		}

		// End marker detection (find LAST occurrence after start marker)
		if startMarkerFound && strings.Contains(strings.TrimSpace(line), strings.TrimSpace(p.markers.EndMarker)) {
			positions.EndMarkerPos = lineNum
		}

		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Validate that we found all required markers
	if positions.StartMarkerBegin == -1 {
		return nil, fmt.Errorf("start marker not found: %v", p.markers.StartMarker)
	}

	if positions.EndMarkerPos == -1 {
		return nil, fmt.Errorf("end marker '%s' not found after line %d", p.markers.EndMarker, positions.StartMarkerEnd)
	}

	if positions.StartMarkerEnd >= positions.EndMarkerPos {
		return nil, errors.New("invalid marker positions: start marker ends after or at end marker")
	}

	// Store the last coordinates found
	if lastPrintX != nil {
		positions.LastPrintX = *lastPrintX
	}
	if lastPrintY != nil {
		positions.LastPrintY = *lastPrintY
	}
	if lastPrintZ != nil {
		positions.LastPrintZ = *lastPrintZ
	}

	return positions, nil
}

// parseGCodeLine parses a G-code line and extracts coordinates
func (p *StreamingProcessor) parseGCodeLine(line string) *GCodeCoordinates {
	// Trim and check if it's a G1 command
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "G1") {
		return nil
	}

	// Regular expressions for extracting coordinates
	xRegex := regexp.MustCompile(`X([-+]?\d*\.?\d+)`)
	yRegex := regexp.MustCompile(`Y([-+]?\d*\.?\d+)`)
	zRegex := regexp.MustCompile(`Z([-+]?\d*\.?\d+)`)
	eRegex := regexp.MustCompile(`E([-+]?\d*\.?\d+)`)

	coords := &GCodeCoordinates{}

	// Extract X coordinate
	if match := xRegex.FindStringSubmatch(trimmed); match != nil {
		if val, err := strconv.ParseFloat(match[1], 64); err == nil {
			coords.X = &val
		}
	}

	// Extract Y coordinate
	if match := yRegex.FindStringSubmatch(trimmed); match != nil {
		if val, err := strconv.ParseFloat(match[1], 64); err == nil {
			coords.Y = &val
		}
	}

	// Extract Z coordinate
	if match := zRegex.FindStringSubmatch(trimmed); match != nil {
		if val, err := strconv.ParseFloat(match[1], 64); err == nil {
			coords.Z = &val
		}
	}

	// Extract E coordinate
	if match := eRegex.FindStringSubmatch(trimmed); match != nil {
		if val, err := strconv.ParseFloat(match[1], 64); err == nil {
			coords.E = &val
		}
	}

	// Return coordinates if we found any
	if coords.X != nil || coords.Y != nil || coords.Z != nil || coords.E != nil {
		return coords
	}

	return nil
}

// findStartMarkerPositions finds multiline start marker using sliding window
func (p *StreamingProcessor) findStartMarkerPositions(filePath string) (int64, int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := int64(0)

	// Sliding window to match multiline patterns
	window := make([]string, 0, len(p.markers.StartMarker)+10) // Small buffer

	for scanner.Scan() {
		line := scanner.Text()
		window = append(window, line)

		// Keep window size reasonable
		maxWindowSize := len(p.markers.StartMarker) + 10
		if len(window) > maxWindowSize {
			window = window[1:] // Remove oldest line
		}

		// Try to find start marker pattern in current window
		if matchPos := p.findStartMarkerInWindow(window, lineNum-int64(len(window))+1); matchPos != nil {
			return matchPos.begin, matchPos.end, nil
		}

		lineNum++
	}

	return 0, 0, fmt.Errorf("start marker not found: %v", p.markers.StartMarker)
}

type startMarkerMatch struct {
	begin int64
	end   int64
}

// findStartMarkerInWindow searches for start marker pattern in the sliding window
func (p *StreamingProcessor) findStartMarkerInWindow(window []string, windowStartLine int64) *startMarkerMatch {
	if len(p.markers.StartMarker) == 1 {
		// Single line marker
		for i, line := range window {
			if strings.Contains(strings.TrimSpace(line), strings.TrimSpace(p.markers.StartMarker[0])) {
				pos := windowStartLine + int64(i)
				return &startMarkerMatch{begin: pos, end: pos}
			}
		}
		return nil
	}

	// Multiline marker search
	for startIdx := 0; startIdx < len(window); startIdx++ {
		if match := p.tryMatchMultilineStart(window, startIdx, windowStartLine); match != nil {
			return match
		}
	}
	return nil
}

// tryMatchMultilineStart attempts to match multiline start marker from given position
func (p *StreamingProcessor) tryMatchMultilineStart(window []string, startIdx int, windowStartLine int64) *startMarkerMatch {
	windowIdx := startIdx
	markerIdx := 0
	firstMarkerLine := int64(-1)
	lastMarkerLine := int64(-1)

	for markerIdx < len(p.markers.StartMarker) && windowIdx < len(window) {
		cleanLine := strings.TrimSpace(window[windowIdx])
		cleanMarker := strings.TrimSpace(p.markers.StartMarker[markerIdx])

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

	if markerIdx == len(p.markers.StartMarker) {
		return &startMarkerMatch{begin: firstMarkerLine, end: lastMarkerLine}
	}
	return nil
}

// findLastEndMarkerPosition finds the LAST occurrence of end marker after start position
func (p *StreamingProcessor) findLastEndMarkerPosition(filePath string, searchFromLine int64) (int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	// Skip to the search start position
	scanner := bufio.NewScanner(file)
	lineNum := int64(0)

	// Skip lines until we reach searchFromLine + 1
	for lineNum <= searchFromLine && scanner.Scan() {
		lineNum++
	}

	lastEndMarkerPos := int64(-1)

	// Continue scanning for end markers
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(strings.TrimSpace(line), strings.TrimSpace(p.markers.EndMarker)) {
			lastEndMarkerPos = lineNum
		}
		lineNum++
	}

	if lastEndMarkerPos == -1 {
		return 0, fmt.Errorf("end marker '%s' not found after line %d", p.markers.EndMarker, searchFromLine)
	}

	return lastEndMarkerPos, scanner.Err()
}

// streamLinesRange streams lines from startLine to endLine (inclusive) with marker splitting
func (p *StreamingProcessor) streamLinesRange(filePath string, writer *bufio.Writer, startLine, endLine int64, processMarkerSplit bool) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := int64(0)

	// Skip to start position
	for lineNum < startLine && scanner.Scan() {
		lineNum++
	}

	// Stream the range
	for lineNum <= endLine && scanner.Scan() {
		line := scanner.Text()

		if processMarkerSplit {
			splitLines := p.processLineWithMarkerSplit(line, p.markers.StartMarker)
			for _, splitLine := range splitLines {
				if _, err := fmt.Fprintln(writer, splitLine); err != nil {
					return err
				}
			}
		} else {
			if _, err := fmt.Fprintln(writer, line); err != nil {
				return err
			}
		}

		lineNum++
	}

	return scanner.Err()
}

// streamLinesFromPosition streams all lines from the given position to EOF
func (p *StreamingProcessor) streamLinesFromPosition(filePath string, writer *bufio.Writer, startLine int64) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := int64(0)

	// Skip to start position
	for lineNum < startLine && scanner.Scan() {
		lineNum++
	}

	// Stream from position to EOF
	for scanner.Scan() {
		line := scanner.Text()
		if _, err := fmt.Fprintln(writer, line); err != nil {
			return err
		}
	}

	return scanner.Err()
}

// streamGeneratedContent writes generated content for an iteration
func (p *StreamingProcessor) streamGeneratedContent(writer *bufio.Writer, iteration int64) error {
	lines := p.generateIterationCode(iteration)
	for _, line := range lines {
		if _, err := fmt.Fprintln(writer, line); err != nil {
			return err
		}
	}
	return nil
}

// processLineWithMarkerSplit splits a line if it contains a marker followed by a comment
func (p *StreamingProcessor) processLineWithMarkerSplit(line string, markers []string) []string {
	for _, marker := range markers {
		cleanMarker := strings.TrimSpace(marker)
		if strings.Contains(line, cleanMarker) {
			semicolonPos := strings.Index(line, ";")
			if semicolonPos != -1 {
				before := strings.TrimSpace(line[:semicolonPos])
				after := strings.TrimSpace(line[semicolonPos:])
				if before != "" && after != "" {
					return []string{before, after}
				}
			}
		}
	}
	return []string{line}
}

func (p *StreamingProcessor) generateIterationCode(iteration int64) []string {
	return []string{
		fmt.Sprintf("; Generated code - Iteration %d", iteration),
		fmt.Sprintf("; Generated code - End iteration %d", iteration),
	}
}

func (p *StreamingProcessor) validateInput() error {
	if len(p.markers.StartMarker) == 0 {
		return errors.New("start marker cannot be empty")
	}

	if strings.TrimSpace(p.markers.EndMarker) == "" {
		return errors.New("end marker cannot be empty")
	}

	if p.config.Iterations <= 0 {
		return errors.New("iterations must be positive")
	}

	// Check for marker conflicts
	for _, startLine := range p.markers.StartMarker {
		if strings.Contains(startLine, p.markers.EndMarker) {
			return fmt.Errorf("start marker line '%s' contains end marker '%s'",
				startLine, p.markers.EndMarker)
		}
	}

	return nil
}

// ProcessFile processes a file using the true streaming processor
func ProcessFile(inputPath, outputPath string, config types.ProcessingRequest) error {
	// Create processor with default markers (these should be configurable)
	markers := PositionMarkers{
		StartMarker: []string{"M211 X0 Y0 Z0 ;turn off soft endstop", "M1007 S1"},
		EndMarker:   "M625",
	}

	processor := NewStreamingProcessor(config, markers)
	return processor.ProcessFile(inputPath, outputPath)
}
