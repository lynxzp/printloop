package main

import "time"

// ProcessingRequest represents a file processing request
type ProcessingRequest struct {
	InputFile  string // Path to input file
	Iterations int64
	WaitTemp   int64
	WaitMin    int64
	Timestamp  int64
}

// ProcessingResult represents the result of file processing
type ProcessingResult struct {
	OutputFile  string    // Output filename
	Data        []byte    // Processed data
	Summary     string    // Brief description of result
	ProcessedAt time.Time // Processing time
	Error       error     // Error if occurred
}

// FileStats contains file statistics
type FileStats struct {
	Lines int `json:"lines"`
	Words int `json:"words"`
	Chars int `json:"characters"`
	Size  int `json:"size_bytes"`
}

// OperationResult contains the result of file operation
type OperationResult struct {
	Operation   string    `json:"operation"`
	InputFile   string    `json:"input_file"`
	OutputFile  string    `json:"output_file"`
	Stats       FileStats `json:"stats"`
	ProcessedAt string    `json:"processed_at"`
	Success     bool      `json:"success"`
	Message     string    `json:"message"`
}
