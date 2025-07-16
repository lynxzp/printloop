package processor

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"printloop/internal/types"
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
	StartMarkerBegin int64 // First line of start marker (0-based)
	StartMarkerEnd   int64 // Last line of start marker (0-based)
	EndMarkerPos     int64 // Position of end marker (0-based)
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

	// Pass 1: Find marker positions
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

// findMarkerPositions uses multiple passes to find marker positions without loading entire file
func (p *StreamingProcessor) findMarkerPositions(filePath string) (*MarkerPositions, error) {
	// Pass 1: Find start marker positions
	startBegin, startEnd, err := p.findStartMarkerPositions(filePath)
	if err != nil {
		return nil, err
	}

	// Pass 2: Find last end marker position
	endPos, err := p.findLastEndMarkerPosition(filePath, startEnd)
	if err != nil {
		return nil, err
	}

	if startEnd >= endPos {
		return nil, errors.New("invalid marker positions: start marker ends after or at end marker")
	}

	return &MarkerPositions{
		StartMarkerBegin: startBegin,
		StartMarkerEnd:   startEnd,
		EndMarkerPos:     endPos,
	}, nil
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
		StartMarker: []string{"M1007 S1"},
		EndMarker:   "G625",
	}

	processor := NewStreamingProcessor(config, markers)
	return processor.ProcessFile(inputPath, outputPath)
}
