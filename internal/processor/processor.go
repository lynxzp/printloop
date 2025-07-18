package processor

import (
	"bufio"
	"embed"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/BurntSushi/toml"
)

// PrinterDefinition represents the complete printer configuration from TOML file
type PrinterDefinition struct {
	Name    string `toml:"name"`
	Markers struct {
		EndInitSection  []string `toml:"endInitSection"`
		EndPrintSection string   `toml:"endPrintSection"`
	} `toml:"markers"`
	SearchStrategy struct {
		EndInitSectionStrategy  string `toml:"endInitSectionStrategy"`
		EndPrintSectionStrategy string `toml:"endPrintSectionStrategy"`
	} `toml:"searchStrategy"`
	Parameters map[string]interface{} `toml:"parameters"`
	Template   struct {
		Code string `toml:"code"`
	} `toml:"template"`
}

// PositionMarkers struct for backward compatibility
type PositionMarkers struct {
	EndInitSection  []string
	EndPrintSection string
}

// SearchStrategy interface for different marker search strategies
type SearchStrategy interface {
	FindInitSectionPosition(filePath string, markers []string) (int64, int64, error)
	FindPrintSectionPosition(filePath string, marker string, searchFromLine int64) (int64, error)
}

// ProcessingRequest represents a file processing request
type ProcessingRequest struct {
	FileName     string
	Iterations   int64
	WaitTemp     int64
	WaitMin      int64
	ExtraExtrude float64
	Printer      string
}

// FirstAppearStrategy finds the first appearance of markers
type FirstAppearStrategy struct{}

// LastAppearStrategy finds the last appearance of markers
type LastAppearStrategy struct{}

func (s *FirstAppearStrategy) FindInitSectionPosition(filePath string, markers []string) (int64, int64, error) {
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

	return 0, 0, fmt.Errorf("start marker not found: %v", markers)
}

func (s *FirstAppearStrategy) FindPrintSectionPosition(filePath string, marker string, searchFromLine int64) (int64, error) {
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

	return 0, fmt.Errorf("end marker '%s' not found after line %d", marker, searchFromLine)
}

func (s *LastAppearStrategy) FindInitSectionPosition(filePath string, markers []string) (int64, int64, error) {
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

func (s *LastAppearStrategy) FindPrintSectionPosition(filePath string, marker string, searchFromLine int64) (int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := int64(0)
	lastFoundPos := int64(-1)

	// Skip to the search start position
	for lineNum <= searchFromLine && scanner.Scan() {
		lineNum++
	}

	// Find last occurrence after searchFromLine
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(strings.TrimSpace(line), strings.TrimSpace(marker)) {
			lastFoundPos = lineNum
		}
		lineNum++
	}

	if lastFoundPos == -1 {
		return 0, fmt.Errorf("end marker '%s' not found after line %d", marker, searchFromLine)
	}

	return lastFoundPos, nil
}

// Factory function to create search strategies
func CreateSearchStrategy(strategyName string) (SearchStrategy, error) {
	switch strategyName {
	case "first_appear":
		return &FirstAppearStrategy{}, nil
	case "last_appear":
		return &LastAppearStrategy{}, nil
	default:
		return nil, fmt.Errorf("unknown search strategy: %s", strategyName)
	}
}

type StreamingProcessor struct {
	config        ProcessingRequest
	printerDef    PrinterDefinition
	initStrategy  SearchStrategy
	printStrategy SearchStrategy
	template      *template.Template
	positions     MarkerPositions
}

// MarkerPositions represents the found positions of start and end markers
type MarkerPositions struct {
	EndInitSectionFirstLine int64   // First line of start marker (0-based)
	EndInitSectionLastLine  int64   // Last line of start marker (0-based)
	EndPrintSection         int64   // Position of end marker (0-based)
	LastPrintX              float64 // X coordinate from last print command (G1 with positive E)
	LastPrintY              float64 // Y coordinate from last print command (G1 with positive E)
	LastPrintZ              float64 // Z coordinate that was active during last print command
}

// GCodeCoordinates holds parsed G-code coordinates
type GCodeCoordinates struct {
	X *float64
	Y *float64
	Z *float64
	E *float64
}

type startMarkerMatch struct {
	begin int64
	end   int64
}

func isValidPrinterName(name string) bool {
	if len(name) == 0 {
		return false
	}
	for _, r := range name {
		isLetter := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		isDigit := r >= '0' && r <= '9'
		isSpecial := r == '-'

		if !(isLetter || isDigit || isSpecial) {
			return false
		}
	}
	return true
}

func NewStreamingProcessor(config ProcessingRequest) (*StreamingProcessor, error) {
	printerName := config.Printer
	// Normalize printer name
	printerName = strings.Replace(printerName, " ", "-", -1)
	printerName = strings.ToLower(printerName)
	// security validate printer name
	if !isValidPrinterName(printerName) {
		return nil, fmt.Errorf("invalid printer name: %s", printerName)
	}

	// Load printer definition from TOML file
	printerDef, err := loadPrinterDefinition(printerName)
	if err != nil {
		return nil, fmt.Errorf("failed to load printer definition: %w", err)
	}

	// Create search strategies
	initStrategy, err := CreateSearchStrategy(printerDef.SearchStrategy.EndInitSectionStrategy)
	if err != nil {
		return nil, fmt.Errorf("failed to create init section strategy: %w", err)
	}

	printStrategy, err := CreateSearchStrategy(printerDef.SearchStrategy.EndPrintSectionStrategy)
	if err != nil {
		return nil, fmt.Errorf("failed to create print section strategy: %w", err)
	}

	// Parse template
	tmpl, err := template.New("printer").Funcs(template.FuncMap{
		"add": func(a, b float64) float64 { return a + b },
		"sub": func(a, b float64) float64 { return a - b },
		"mul": func(a, b int) int { return a * b },
		"max": func(a, b float64) float64 {
			if a > b {
				return a
			}
			return b
		},
	}).Parse(printerDef.Template.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return &StreamingProcessor{
		config:        config,
		printerDef:    *printerDef,
		initStrategy:  initStrategy,
		printStrategy: printStrategy,
		template:      tmpl,
	}, nil
}

//go:embed printers/*.toml
var printerConfigs embed.FS

func loadPrinterDefinition(printerName string) (*PrinterDefinition, error) {
	filename := "printers/" + printerName + ".toml"
	data, err := printerConfigs.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var def PrinterDefinition
	err = toml.Unmarshal(data, &def)
	return &def, err
}

// ProcessFile processes a file using true streaming with multiple passes
func (p *StreamingProcessor) ProcessFile(inputPath, outputPath string) error {
	// Validate input first
	if err := p.validateInput(); err != nil {
		return err
	}

	// Pass 1: Find marker positions and extract G-code coordinates
	var err error
	pos, err := p.findMarkerPositions(inputPath)
	if err != nil {
		return err
	}
	p.positions = *pos

	// Open output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	writer := bufio.NewWriter(outputFile)
	defer writer.Flush()

	// Pass 2: Stream header (lines 0 to EndInitSectionLastLine inclusive)
	if err := p.streamLinesRange(inputPath, writer, 0, p.positions.EndInitSectionLastLine, true); err != nil {
		return fmt.Errorf("failed to stream header: %w", err)
	}

	// Pass 3: For each iteration, stream body + end marker + generated content
	for i := int64(0); i < p.config.Iterations; i++ {
		// Stream body (lines after EndInitSectionLastLine to before EndPrintSection)
		if p.positions.EndInitSectionLastLine+1 < p.positions.EndPrintSection {
			if err := p.streamLinesRange(inputPath, writer, p.positions.EndInitSectionLastLine+1, p.positions.EndPrintSection-1, false); err != nil {
				return fmt.Errorf("failed to stream body for iteration %d: %w", i+1, err)
			}
		}

		// Stream end marker line
		if err := p.streamLinesRange(inputPath, writer, p.positions.EndPrintSection, p.positions.EndPrintSection, false); err != nil {
			return fmt.Errorf("failed to stream end marker for iteration %d: %w", i+1, err)
		}

		// Stream generated content
		if err := p.streamGeneratedContent(writer, i+1); err != nil {
			return fmt.Errorf("failed to stream generated content for iteration %d: %w", i+1, err)
		}
	}

	// Pass 4: Stream footer (lines after EndPrintSection to EOF)
	if err := p.streamLinesFromPosition(inputPath, writer, p.positions.EndPrintSection+1); err != nil {
		return fmt.Errorf("failed to stream footer: %w", err)
	}

	return nil
}

// findMarkerPositions uses strategies to find marker positions and extract G-code coordinates
func (p *StreamingProcessor) findMarkerPositions(filePath string) (*MarkerPositions, error) {
	// Find init section positions using strategy
	initFirst, initLast, err := p.initStrategy.FindInitSectionPosition(filePath, p.printerDef.Markers.EndInitSection)
	if err != nil {
		return nil, err
	}

	// Find print section position using strategy
	printPos, err := p.printStrategy.FindPrintSectionPosition(filePath, p.printerDef.Markers.EndPrintSection, initLast)
	if err != nil {
		return nil, err
	}

	if initLast >= printPos {
		return nil, errors.New("invalid marker positions: start marker ends after or at end marker")
	}

	// Extract G-code coordinates
	lastPrintX, lastPrintY, lastPrintZ, err := p.extractGCodeCoordinates(filePath)
	if err != nil {
		return nil, err
	}

	positions := &MarkerPositions{
		EndInitSectionFirstLine: initFirst,
		EndInitSectionLastLine:  initLast,
		EndPrintSection:         printPos,
		LastPrintX:              lastPrintX,
		LastPrintY:              lastPrintY,
		LastPrintZ:              lastPrintZ,
	}

	return positions, nil
}

// extractGCodeCoordinates scans file and extracts last print coordinates
func (p *StreamingProcessor) extractGCodeCoordinates(filePath string) (float64, float64, float64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lastPrintX, lastPrintY, lastPrintZ *float64
	var currentZ *float64

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
	}

	if err := scanner.Err(); err != nil {
		return 0, 0, 0, err
	}

	// Return coordinates with defaults if not found
	var x, y, z float64
	if lastPrintX != nil {
		x = *lastPrintX
	}
	if lastPrintY != nil {
		y = *lastPrintY
	}
	if lastPrintZ != nil {
		z = *lastPrintZ
	}

	return x, y, z, nil
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
			splitLines := p.processLineWithMarkerSplit(line, p.printerDef.Markers.EndInitSection)
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

// streamGeneratedContent writes generated content for an iteration using template
func (p *StreamingProcessor) streamGeneratedContent(writer *bufio.Writer, iteration int64) error {
	// Prepare template data
	templateData := struct {
		PrinterName string
		Iteration   int64
		Request     ProcessingRequest
		Config      map[string]interface{}
		Positions   MarkerPositions
	}{
		PrinterName: p.printerDef.Name,
		Iteration:   iteration,
		Request:     p.config,
		Config:      p.printerDef.Parameters,
		Positions:   p.positions,
	}

	// Execute template
	var output strings.Builder
	if err := p.template.Execute(&output, templateData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Write generated content
	lines := strings.Split(output.String(), "\n")
	for _, line := range lines {
		if line != "" || len(lines) == 1 { // Don't write empty lines unless it's the only line
			if _, err := fmt.Fprintln(writer, line); err != nil {
				return err
			}
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

func (p *StreamingProcessor) validateInput() error {
	if len(p.printerDef.Markers.EndInitSection) == 0 {
		return errors.New("EndInitSection marker cannot be empty")
	}

	if strings.TrimSpace(p.printerDef.Markers.EndPrintSection) == "" {
		return errors.New("EndPrintSection marker cannot be empty")
	}

	if p.config.Iterations <= 0 {
		return errors.New("iterations must be positive")
	}

	// Check for marker conflicts
	for _, startLine := range p.printerDef.Markers.EndInitSection {
		if strings.Contains(startLine, p.printerDef.Markers.EndPrintSection) {
			return fmt.Errorf("EndInitSection marker line '%s' contains EndPrintSection marker '%s'",
				startLine, p.printerDef.Markers.EndPrintSection)
		}
	}

	return nil
}

// ProcessFile processes a file using the true streaming processor with printer configuration
func ProcessFile(inputPath, outputPath string, config ProcessingRequest) error {
	processor, err := NewStreamingProcessor(config)
	if err != nil {
		return err
	}

	return processor.ProcessFile(inputPath, outputPath)
}
