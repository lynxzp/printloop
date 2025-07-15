// processor.go - Refactored with separated concerns
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
	StartMarker string // e.g., "START_PRINT"
	EndMarker   string // e.g., "END_PRINT"
}

func (p *GCodeProcessor) ProcessLines(lines []string, markers PositionMarkers) (*ProcessingResult, error) {
	startMarkerPos, endMarkerPos, err := p.findMarkerPositions(lines, markers)
	if err != nil {
		return nil, err
	}

	result := &ProcessingResult{}

	// Copy header (before start marker, including the start marker)
	result.Lines = append(result.Lines, lines[:startMarkerPos+1]...)

	// Process body with iterations (between markers, excluding both markers)
	bodyLines := lines[startMarkerPos+1 : endMarkerPos]
	endMarkerLine := lines[endMarkerPos]

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
	result.Lines = append(result.Lines, lines[endMarkerPos+1:]...)

	return result, nil
}

func (p *GCodeProcessor) findMarkerPositions(lines []string, markers PositionMarkers) (int, int, error) {
	startMarkerPos := -1
	endMarkerPos := -1

	for i, line := range lines {
		cleanLine := strings.TrimSpace(line)
		if strings.Contains(cleanLine, markers.StartMarker) {
			startMarkerPos = i // Position OF start marker
		}
		if strings.Contains(cleanLine, markers.EndMarker) {
			endMarkerPos = i // Position OF end marker
		}
	}

	if startMarkerPos == -1 {
		return 0, 0, errors.New("start marker not found")
	}
	if endMarkerPos == -1 {
		return 0, 0, errors.New("end marker not found")
	}
	if startMarkerPos >= endMarkerPos {
		return 0, 0, errors.New("invalid marker positions")
	}

	return startMarkerPos, endMarkerPos, nil
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
		StartMarker: "M1007 S1",
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
