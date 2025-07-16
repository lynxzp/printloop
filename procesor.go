// file: processor.go
// processor.go - Refactored with multiline start marker support
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// GCodeProcessor ??? Core business logic separated from I/O
type GCodeProcessor struct {
	config ProcessingRequest
}

type ProcessingResult struct {
	Lines []string
}

type PositionMarkers struct {
	StartMarker []string // For multiline markers, each element is a line
	EndMarker   string   // Keep single line for now
}

// MarkerPositions represents the found positions of start and end markers
type MarkerPositions struct {
	StartMarkerBegin int // First line of start marker
	StartMarkerEnd   int // Last line of start marker
	EndMarkerPos     int // Position of end marker
}

func (p *GCodeProcessor) ProcessLines(lines []string, markers PositionMarkers) (*ProcessingResult, error) {
	positions, err := p.findMarkerPositions(lines, markers)
	if err != nil {
		return nil, err
	}

	result := &ProcessingResult{}

	// Copy header (before start marker, including the complete start marker)
	for i := 0; i <= positions.StartMarkerEnd; i++ {
		splitLines := p.processLineWithMarkerSplit(lines[i], markers.StartMarker)
		result.Lines = append(result.Lines, splitLines...)
	}

	// Process body with iterations (between end of start marker and end marker)
	bodyLines := lines[positions.StartMarkerEnd+1 : positions.EndMarkerPos]
	endMarkerLine := lines[positions.EndMarkerPos]

	for i := int64(0); i < p.config.Iterations; i++ {
		// Add body content
		result.Lines = append(result.Lines, bodyLines...)
		// Add end marker for this iteration
		result.Lines = append(result.Lines, endMarkerLine)
		// Add generated content
		generated := p.generateIterationCode(i + 1)
		result.Lines = append(result.Lines, generated...)
	}

	// Copy footer (after end marker, excluding the end marker)
	result.Lines = append(result.Lines, lines[positions.EndMarkerPos+1:]...)

	return result, nil
}

// processLineWithMarkerSplit splits a line if it contains a marker followed by a comment
func (p *GCodeProcessor) processLineWithMarkerSplit(line string, markers []string) []string {
	// Check if this line contains any of the markers and has a semicolon
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

func (p *GCodeProcessor) findMarkerPositions(lines []string, markers PositionMarkers) (*MarkerPositions, error) {
	if len(markers.StartMarker) == 0 {
		return nil, errors.New("start marker cannot be empty")
	}

	startMarkerBegin, startMarkerEnd, err := p.findMultilineStartMarker(lines, markers.StartMarker)
	if err != nil {
		return nil, err
	}

	endMarkerPos, err := p.findSingleLineEndMarker(lines, markers.EndMarker, startMarkerEnd)
	if err != nil {
		return nil, err
	}

	if startMarkerEnd >= endMarkerPos {
		return nil, errors.New("invalid marker positions: start marker ends after or at end marker")
	}

	return &MarkerPositions{
		StartMarkerBegin: startMarkerBegin,
		StartMarkerEnd:   startMarkerEnd,
		EndMarkerPos:     endMarkerPos,
	}, nil
}

func (p *GCodeProcessor) findMultilineStartMarker(lines []string, startMarkerLines []string) (int, int, error) {
	if len(startMarkerLines) == 1 {
		// Single line marker - backward compatibility
		return p.findSingleLineStartMarker(lines, startMarkerLines[0])
	}

	// Multiline marker search with support for empty/comment lines
	for startIdx := 0; startIdx <= len(lines)-1; startIdx++ {
		// Try to match starting from startIdx
		lineIdx := startIdx
		markerIdx := 0
		firstMarkerIdx := -1

		for markerIdx < len(startMarkerLines) && lineIdx < len(lines) {
			cleanLine := strings.TrimSpace(lines[lineIdx])
			cleanMarker := strings.TrimSpace(startMarkerLines[markerIdx])

			if strings.Contains(cleanLine, cleanMarker) {
				// Found this marker line
				if firstMarkerIdx == -1 {
					firstMarkerIdx = lineIdx
				}
				markerIdx++
				lineIdx++
			} else if cleanLine == "" || strings.HasPrefix(cleanLine, ";") {
				// Skip empty or comment lines
				lineIdx++
			} else {
				// This line doesn't match and isn't skippable, so this start position doesn't work
				break
			}
		}

		if markerIdx == len(startMarkerLines) {
			// Found all marker lines
			return firstMarkerIdx, lineIdx - 1, nil
		}
	}

	return 0, 0, errors.New("multiline start marker not found")
}

func (p *GCodeProcessor) findSingleLineStartMarker(lines []string, startMarker string) (int, int, error) {
	for i, line := range lines {
		cleanLine := strings.TrimSpace(line)
		if strings.Contains(cleanLine, startMarker) {
			return i, i, nil // Same position for single line
		}
	}
	return 0, 0, errors.New("single line start marker not found")
}

func (p *GCodeProcessor) findSingleLineEndMarker(lines []string, endMarker string, searchFromIndex int) (int, error) {
	endMarkerPos := -1

	// Search from after the start marker to end of file, keeping track of LAST occurrence
	for i := searchFromIndex + 1; i < len(lines); i++ {
		cleanLine := strings.TrimSpace(lines[i])
		if strings.Contains(cleanLine, endMarker) {
			endMarkerPos = i // Keep updating to find the LAST occurrence
		}
	}

	if endMarkerPos == -1 {
		return 0, errors.New("end marker not found")
	}

	return endMarkerPos, nil
}

func (p *GCodeProcessor) generateIterationCode(iteration int64) []string {
	var result []string

	result = append(result, fmt.Sprintf("; Generated code - Iteration %d", iteration))
	result = append(result, fmt.Sprintf("; Generated code - End iteration %d", iteration))

	return result
}

// Utility functions for working with io.ReadWriteSeeker
func readLinesFromReader(reader io.Reader) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func writeLinesToWriter(writer io.Writer, lines []string) error {
	for _, line := range lines {
		if _, err := fmt.Fprintln(writer, line); err != nil {
			return err
		}
	}
	return nil
}

func ProcessWithReaderWriter(reader io.ReadSeeker, writer io.Writer, config ProcessingRequest) error {
	// Read lines from reader
	lines, err := readLinesFromReader(reader)
	if err != nil {
		return fmt.Errorf("failed to read lines: %w", err)
	}

	// Process with core logic
	processor := &GCodeProcessor{config: config}
	markers := PositionMarkers{
		StartMarker: []string{"M1007 S1"}, // Convert to slice for consistency
		EndMarker:   "G625",
	}

	result, err := processor.ProcessLines(lines, markers)
	if err != nil {
		return fmt.Errorf("failed to process: %w", err)
	}

	// Write result to writer
	if err := writeLinesToWriter(writer, result.Lines); err != nil {
		return fmt.Errorf("failed to write lines: %w", err)
	}

	return nil
}

func ProcessFile(inputPath, outputPath string, config ProcessingRequest) error {
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer inputFile.Close()

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	return ProcessWithReaderWriter(inputFile, outputFile, config)
}
