package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// ProcessFile processes a file according to the request
func ProcessFile(request ProcessingRequest) (*ProcessingResult, error) {
	// Read input file
	content, err := os.ReadFile(request.InputFile)
	if err != nil {
		return nil, fmt.Errorf("file reading error: %w", err)
	}

	// Perform operation
	processedData, stats, err := performOperation(string(content), request.Operation)
	if err != nil {
		return nil, fmt.Errorf("operation execution error: %w", err)
	}

	// Format output
	formattedData, err := formatOutput(processedData, stats, request.Format, request.Operation)
	if err != nil {
		return nil, fmt.Errorf("formatting error: %w", err)
	}

	// Create result summary
	summary := createSummary(request.Operation, stats, len(formattedData))

	result := &ProcessingResult{
		OutputFile:  fmt.Sprintf("result_%d.%s", request.Timestamp, request.Format),
		Data:        formattedData,
		Summary:     summary,
		ProcessedAt: time.Now(),
		Error:       nil,
	}

	// Clean up input file
	os.Remove(request.InputFile)

	return result, nil
}

// performOperation executes the specified operation on text
func performOperation(content, operation string) (string, FileStats, error) {
	stats := calculateStats(content)

	switch operation {
	case "uppercase":
		return strings.ToUpper(content), stats, nil

	case "lowercase":
		return strings.ToLower(content), stats, nil

	case "word_count":
		result := fmt.Sprintf("Word count: %d\nLine count: %d\nCharacter count: %d",
			stats.Words, stats.Lines, stats.Chars)
		return result, stats, nil

	case "line_count":
		result := fmt.Sprintf("Line count: %d", stats.Lines)
		return result, stats, nil

	case "reverse":
		return reverseString(content), stats, nil

	default:
		return "", stats, fmt.Errorf("unknown operation: %s", operation)
	}
}

// calculateStats calculates text statistics
func calculateStats(content string) FileStats {
	lines := len(strings.Split(content, "\n"))
	words := len(strings.Fields(content))
	chars := len(content)
	size := len([]byte(content))

	return FileStats{
		Lines: lines,
		Words: words,
		Chars: chars,
		Size:  size,
	}
}

// reverseString reverses a string
func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// formatOutput formats the result to the specified format
func formatOutput(data string, stats FileStats, format, operation string) ([]byte, error) {
	switch format {
	case "txt":
		return []byte(data), nil

	case "json":
		result := OperationResult{
			Operation:   operation,
			Stats:       stats,
			ProcessedAt: time.Now().Format("2006-01-02 15:04:05"),
			Success:     true,
			Message:     data,
		}
		return json.MarshalIndent(result, "", "  ")

	case "csv":
		return formatAsCSV(data, stats, operation)

	default:
		return nil, fmt.Errorf("unknown format: %s", format)
	}
}

// formatAsCSV formats result as CSV
func formatAsCSV(data string, stats FileStats, operation string) ([]byte, error) {
	var records [][]string

	// Headers
	records = append(records, []string{"Parameter", "Value"})

	// Main data
	records = append(records, []string{"Operation", operation})
	records = append(records, []string{"Lines", strconv.Itoa(stats.Lines)})
	records = append(records, []string{"Words", strconv.Itoa(stats.Words)})
	records = append(records, []string{"Characters", strconv.Itoa(stats.Chars)})
	records = append(records, []string{"Size (bytes)", strconv.Itoa(stats.Size)})
	records = append(records, []string{"Processed At", time.Now().Format("2006-01-02 15:04:05")})

	// If data is short, add it
	if len(data) < 1000 {
		// Replace line breaks for CSV
		cleanData := strings.ReplaceAll(data, "\n", "\\n")
		cleanData = strings.ReplaceAll(cleanData, "\r", "\\r")
		records = append(records, []string{"Result", cleanData})
	} else {
		records = append(records, []string{"Result", "Data too long for CSV (saved in separate file)"})
	}

	// Convert to CSV
	var output strings.Builder
	writer := csv.NewWriter(&output)

	for _, record := range records {
		if err := writer.Write(record); err != nil {
			return nil, fmt.Errorf("CSV write error: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("CSV finalization error: %w", err)
	}

	return []byte(output.String()), nil
}

// createSummary creates a brief description of the result
func createSummary(operation string, stats FileStats, resultSize int) string {
	var operationDesc string

	switch operation {
	case "uppercase":
		operationDesc = "Text converted to uppercase"
	case "lowercase":
		operationDesc = "Text converted to lowercase"
	case "word_count":
		operationDesc = "Text statistics calculated"
	case "line_count":
		operationDesc = "Line count calculated"
	case "reverse":
		operationDesc = "Text reversed"
	default:
		operationDesc = "Operation performed: " + operation
	}

	return fmt.Sprintf("%s. Processed %d words in %d lines. Result size: %d bytes.",
		operationDesc, stats.Words, stats.Lines, resultSize)
}
