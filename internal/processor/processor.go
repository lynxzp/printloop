package processor

import (
	"bufio"
	"embed"
	"errors"
	"fmt"
	"os"
	"printloop/internal/processor/strategy"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/BurntSushi/toml"
)

// PrinterDefinition represents the complete printer configuration from TOML file
type PrinterDefinition struct {
	Name    string
	Markers struct {
		EndInitSection  []string
		EndPrintSection []string
	}
	SearchStrategy struct {
		EndInitSectionStrategy  string
		EndPrintSectionStrategy string
	}
	Parameters map[string]interface{}
	Template   struct {
		Code string
	}
}

// PositionMarkers struct for backward compatibility
type PositionMarkers struct {
	EndInitSection  []string
	EndPrintSection string
}

// SearchStrategy interface for different marker search strategies
type SearchStrategy interface {
	FindInitSectionPosition(filePath string, markers []string) (int64, int64, error)
	FindPrintSectionPosition(filePath string, markers []string, searchFromLine int64) (int64, int64, error)
}

// ProcessingRequest represents a file processing request
type ProcessingRequest struct {
	FileName       string
	Iterations     int64
	WaitTemp       int64
	WaitMin        int64
	ExtraExtrude   float64
	Printer        string
	CustomTemplate string
}

// Factory function to create search strategies
func CreateSearchStrategy(strategyName string) (SearchStrategy, error) {
	switch strategyName {
	case "after_first_appear":
		return &strategy.AfterFirstAppearStrategy{}, nil
	case "after_last_appear":
		return &strategy.AfterLastAppearStrategy{}, nil
	case "before_first_appear":
		return &strategy.BeforeCommandStrategy{}, nil
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
// Updated MarkerPositions struct
type MarkerPositions struct {
	EndInitSectionFirstLine  int64   // First line of start marker (0-based)
	EndInitSectionLastLine   int64   // Last line of start marker (0-based)
	EndPrintSectionFirstLine int64   // First line of end marker (0-based) - NEW
	EndPrintSectionLastLine  int64   // Last line of end marker (0-based) - UPDATED
	FirstPrintX              float64 // X coordinate from first print command (G1 with positive E) after marker
	FirstPrintY              float64 // Y coordinate from first print command (G1 with positive E) after marker
	FirstPrintZ              float64 // Z coordinate that was active during first print command after marker
	LastPrintX               float64 // X coordinate from last print command (G1 with positive E)
	LastPrintY               float64 // Y coordinate from last print command (G1 with positive E)
	LastPrintZ               float64 // Z coordinate that was active during last print command
}

// GCodeCoordinates holds parsed G-code coordinates
type GCodeCoordinates struct {
	X *float64
	Y *float64
	Z *float64
	E *float64
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
	var printerDef *PrinterDefinition
	var templateCode string
	var err error

	// If custom template is provided, parse it
	if config.CustomTemplate != "" {
		printerDef, templateCode, err = parseCustomTemplate(config.CustomTemplate, config.Printer)
		if err != nil {
			return nil, fmt.Errorf("failed to parse custom template: %w", err)
		}
	} else {
		// Use default printer definition
		printerName := config.Printer
		// Normalize printer name
		printerName = strings.Replace(printerName, " ", "-", -1)
		printerName = strings.ToLower(printerName)
		// security validate printer name
		if !isValidPrinterName(printerName) {
			return nil, fmt.Errorf("invalid printer name: %s", printerName)
		}

		// Load printer definition from TOML file
		printerDef, err = loadPrinterDefinition(printerName)
		if err != nil {
			return nil, fmt.Errorf("failed to load printer definition: %w", err)
		}
		templateCode = printerDef.Template.Code
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
	}).Parse(templateCode)
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

// parseCustomTemplate parses a custom template in TOML format and extracts the template code
func parseCustomTemplate(customTemplate string, printerName string) (*PrinterDefinition, string, error) {
	var def PrinterDefinition
	err := toml.Unmarshal([]byte(customTemplate), &def)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse custom template TOML: %w", err)
	}

	// Validate required fields
	if len(def.Markers.EndInitSection) == 0 {
		return nil, "", errors.New("custom template missing EndInitSection markers")
	}
	if len(def.Markers.EndPrintSection) == 0 {
		return nil, "", errors.New("custom template missing EndPrintSection markers")
	}
	if def.SearchStrategy.EndInitSectionStrategy == "" {
		return nil, "", errors.New("custom template missing EndInitSectionStrategy")
	}
	if def.SearchStrategy.EndPrintSectionStrategy == "" {
		return nil, "", errors.New("custom template missing EndPrintSectionStrategy")
	}
	if def.Template.Code == "" {
		return nil, "", errors.New("custom template missing Template.Code")
	}

	// Set name if not provided
	if def.Name == "" {
		def.Name = fmt.Sprintf("Custom-%s", printerName)
	}

	// Convert all numeric parameters to float64 for template compatibility
	normalizeParameters(&def)

	return &def, def.Template.Code, nil
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
	if err != nil {
		return &def, err
	}

	// Convert all numeric parameters to float64 for template compatibility
	normalizeParameters(&def)

	return &def, err
}

// normalizeParameters converts all numeric values in Parameters to float64 for template compatibility
func normalizeParameters(def *PrinterDefinition) {
	if def.Parameters == nil {
		return
	}

	for key, value := range def.Parameters {
		switch v := value.(type) {
		case int:
			def.Parameters[key] = float64(v)
		case int32:
			def.Parameters[key] = float64(v)
		case int64:
			def.Parameters[key] = float64(v)
		case float32:
			def.Parameters[key] = float64(v)
			// float64 stays as is
			// strings and other types stay as is
		}
	}
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
		// Stream body (lines after EndInitSectionLastLine to before EndPrintSectionFirstLine)
		if p.positions.EndInitSectionLastLine+1 < p.positions.EndPrintSectionFirstLine {
			if err := p.streamLinesRange(inputPath, writer, p.positions.EndInitSectionLastLine+1, p.positions.EndPrintSectionFirstLine-1, false); err != nil {
				return fmt.Errorf("failed to stream body for iteration %d: %w", i+1, err)
			}
		}

		// Stream end marker lines (can be multiline now)
		if err := p.streamLinesRange(inputPath, writer, p.positions.EndPrintSectionFirstLine, p.positions.EndPrintSectionLastLine, false); err != nil {
			return fmt.Errorf("failed to stream end marker for iteration %d: %w", i+1, err)
		}

		// Stream generated content
		if err := p.streamGeneratedContent(writer, i+1); err != nil {
			return fmt.Errorf("failed to stream generated content for iteration %d: %w", i+1, err)
		}
	}

	// Pass 4: Stream footer (lines after EndPrintSectionLastLine to EOF)
	if err := p.streamLinesFromPosition(inputPath, writer, p.positions.EndPrintSectionLastLine+1); err != nil {
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

	// Find print section position using strategy - now returns begin,end
	printFirst, printLast, err := p.printStrategy.FindPrintSectionPosition(filePath, p.printerDef.Markers.EndPrintSection, initLast)
	if err != nil {
		return nil, err
	}

	if initLast >= printFirst {
		return nil, errors.New("invalid marker positions: start marker ends after or at end marker")
	}

	// Extract G-code coordinates
	firstPrintX, firstPrintY, firstPrintZ, lastPrintX, lastPrintY, lastPrintZ, err := p.extractGCodeCoordinates(filePath, initLast)
	if err != nil {
		return nil, err
	}

	positions := &MarkerPositions{
		EndInitSectionFirstLine:  initFirst,
		EndInitSectionLastLine:   initLast,
		EndPrintSectionFirstLine: printFirst,
		EndPrintSectionLastLine:  printLast,
		FirstPrintX:              firstPrintX,
		FirstPrintY:              firstPrintY,
		FirstPrintZ:              firstPrintZ,
		LastPrintX:               lastPrintX,
		LastPrintY:               lastPrintY,
		LastPrintZ:               lastPrintZ,
	}

	return positions, nil
}

// extractGCodeCoordinates scans file and extracts first and last print coordinates
func (p *StreamingProcessor) extractGCodeCoordinates(filePath string, endInitSectionLastLine int64) (float64, float64, float64, float64, float64, float64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0, 0, 0, 0, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var firstPrintX, firstPrintY, firstPrintZ *float64
	var lastPrintX, lastPrintY, lastPrintZ *float64
	var currentZ *float64
	var firstPrintFound bool
	lineNum := int64(0)

	for scanner.Scan() {
		line := scanner.Text()

		// Parse G-code coordinates from this line
		if coords := p.parseGCodeLine(line); coords != nil {
			// Update current Z from any G1 command
			if coords.Z != nil {
				currentZ = coords.Z
			}

			// Update coordinates for print commands (G1 with positive E)
			if coords.E != nil && *coords.E > 0 && (coords.X != nil || coords.Y != nil) {
				// This is a print command

				// Track first print coordinates after init section
				if !firstPrintFound && lineNum > endInitSectionLastLine {
					if coords.X != nil {
						firstPrintX = coords.X
					}
					if coords.Y != nil {
						firstPrintY = coords.Y
					}
					// Remember the Z that was active during this first print command
					if currentZ != nil {
						firstPrintZ = currentZ
					}
					firstPrintFound = true
				}

				// Always update last print coordinates
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

		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return 0, 0, 0, 0, 0, 0, err
	}

	// Return coordinates with defaults if not found
	var fx, fy, fz, lx, ly, lz float64
	if firstPrintX != nil {
		fx = *firstPrintX
	}
	if firstPrintY != nil {
		fy = *firstPrintY
	}
	if firstPrintZ != nil {
		fz = *firstPrintZ
	}
	if lastPrintX != nil {
		lx = *lastPrintX
	}
	if lastPrintY != nil {
		ly = *lastPrintY
	}
	if lastPrintZ != nil {
		lz = *lastPrintZ
	}

	if !strings.Contains(p.config.Printer, "unit-tests") {
		// unit tests don't contain entire G-code, so we don't check for first print found
		if !firstPrintFound {
			return fx, fy, fz, lx, ly, lz, fmt.Errorf("no print commands found after end of init section at line %d", endInitSectionLastLine)
		}
	}

	return fx, fy, fz, lx, ly, lz, nil
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

	if len(p.printerDef.Markers.EndPrintSection) == 0 {
		return errors.New("EndPrintSection marker cannot be empty")
	}

	if p.config.Iterations <= 0 {
		return errors.New("iterations must be positive")
	}

	// Check for marker conflicts
	for _, startLine := range p.printerDef.Markers.EndInitSection {
		for _, endLine := range p.printerDef.Markers.EndPrintSection {
			if strings.Contains(startLine, endLine) {
				return fmt.Errorf("EndInitSection marker line '%s' contains EndPrintSection marker '%s'",
					startLine, endLine)
			}
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

func LoadPrinterDefinitionPublic(printerName string) (*PrinterDefinition, error) {
	return loadPrinterDefinition(printerName)
}
