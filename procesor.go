package main

import (
	"os"
	"path"
)

// ProcessFile processes a file according to the request
func ProcessFile(request ProcessingRequest) error {
	inFile := path.Join("uploads", request.FileName)
	outFile := path.Join("results", request.FileName)

	// Simulate file processing
	err := os.Rename(inFile, outFile)
	if err != nil {
		return err
	}

	return nil
}
