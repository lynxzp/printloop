package types

// ProcessingRequest represents a file processing request
type ProcessingRequest struct {
	FileName     string
	Iterations   int64
	WaitTemp     int64
	WaitMin      int64
	ExtraExtrude float64
	Printer      string
}
