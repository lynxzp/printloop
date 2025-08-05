package webserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// ErrorType represents different categories of errors
type ErrorType string

const (
	ErrorTypeFileProcessing   ErrorType = "file_processing"
	ErrorTypeTemplate         ErrorType = "template"
	ErrorTypeValidation       ErrorType = "validation"
	ErrorTypeConfiguration    ErrorType = "configuration"
	ErrorTypeFileIO           ErrorType = "file_io"
	ErrorTypeUpload           ErrorType = "upload"
	ErrorTypeInternal         ErrorType = "internal"
)

// ErrorResponse represents a structured error response
type ErrorResponse struct {
	Type        ErrorType `json:"type"`
	Code        string    `json:"code"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Details     string    `json:"details"`
	Suggestions []string  `json:"suggestions,omitempty"`
}

// CategorizeError analyzes an error and returns an appropriate ErrorResponse
func CategorizeError(err error) ErrorResponse {
	if err == nil {
		return ErrorResponse{
			Type:        ErrorTypeInternal,
			Code:        "unknown_error",
			Title:       "Unknown Error",
			Description: "An unknown error occurred.",
			Details:     "No error details available",
		}
	}

	errMsg := err.Error()
	errMsgLower := strings.ToLower(errMsg)

	// Template-related errors
	if strings.Contains(errMsgLower, "template") || strings.Contains(errMsgLower, "parse") {
		if strings.Contains(errMsgLower, "custom template") {
			return ErrorResponse{
				Type:        ErrorTypeTemplate,
				Code:        "custom_template_error",
				Title:       "Custom Template Error",
				Description: "There was an error processing your custom template.",
				Details:     errMsg,
				Suggestions: []string{
					"Check template syntax for proper TOML format",
					"Ensure all required sections are present (Markers, SearchStrategy, Template)",
					"Validate template variables and functions",
				},
			}
		}
		return ErrorResponse{
			Type:        ErrorTypeTemplate,
			Code:        "template_parsing_error",
			Title:       "Template Parsing Error",
			Description: "The printer template could not be parsed or executed.",
			Details:     errMsg,
			Suggestions: []string{
				"Try selecting a different printer",
				"Check if the printer template is properly configured",
			},
		}
	}

	// File processing errors
	if strings.Contains(errMsgLower, "marker") || strings.Contains(errMsgLower, "position") {
		return ErrorResponse{
			Type:        ErrorTypeFileProcessing,
			Code:        "marker_not_found",
			Title:       "G-code Markers Not Found",
			Description: "Required markers for loop insertion were not found in the G-code file.",
			Details:     errMsg,
			Suggestions: []string{
				"Ensure your G-code contains the required start and end markers",
				"Try a different printer profile that matches your slicer settings",
				"Check if the G-code was generated with compatible slicer settings",
			},
		}
	}

	if strings.Contains(errMsgLower, "print command") || strings.Contains(errMsgLower, "coordinates") {
		return ErrorResponse{
			Type:        ErrorTypeFileProcessing,
			Code:        "invalid_gcode_structure",
			Title:       "Invalid G-code Structure",
			Description: "The G-code file does not contain the expected structure for loop processing.",
			Details:     errMsg,
			Suggestions: []string{
				"Ensure the file contains actual print commands (G1 with positive E values)",
				"Check that the G-code file is complete and not truncated",
				"Verify the file was exported correctly from your slicer",
			},
		}
	}

	// Printer configuration errors
	if strings.Contains(errMsgLower, "printer") {
		if strings.Contains(errMsgLower, "not found") || strings.Contains(errMsgLower, "load") {
			return ErrorResponse{
				Type:        ErrorTypeConfiguration,
				Code:        "printer_not_found",
				Title:       "Printer Configuration Not Found",
				Description: "The selected printer configuration could not be loaded.",
				Details:     errMsg,
				Suggestions: []string{
					"Select a different printer from the dropdown",
					"Use a custom template if your printer is not supported",
				},
			}
		}
		if strings.Contains(errMsgLower, "invalid") {
			return ErrorResponse{
				Type:        ErrorTypeValidation,
				Code:        "invalid_printer_name",
				Title:       "Invalid Printer Name",
				Description: "The printer name contains invalid characters or format.",
				Details:     errMsg,
				Suggestions: []string{
					"Printer names can only contain letters, numbers, and hyphens",
					"Select a printer from the dropdown instead of typing manually",
				},
			}
		}
	}

	// Validation errors
	if strings.Contains(errMsgLower, "iteration") || strings.Contains(errMsgLower, "positive") {
		return ErrorResponse{
			Type:        ErrorTypeValidation,
			Code:        "invalid_parameters",
			Title:       "Invalid Parameters",
			Description: "One or more processing parameters have invalid values.",
			Details:     errMsg,
			Suggestions: []string{
				"Ensure iteration count is a positive number",
				"Check that all numeric values are within valid ranges",
				"Review all form fields for correct input",
			},
		}
	}

	// File I/O errors
	if strings.Contains(errMsgLower, "file") {
		if strings.Contains(errMsgLower, "create") || strings.Contains(errMsgLower, "write") {
			return ErrorResponse{
				Type:        ErrorTypeFileIO,
				Code:        "file_write_error",
				Title:       "File Write Error",
				Description: "Could not create or write the processed file.",
				Details:     errMsg,
				Suggestions: []string{
					"Check server disk space",
					"Try uploading the file again",
				},
			}
		}
		if strings.Contains(errMsgLower, "open") || strings.Contains(errMsgLower, "read") {
			return ErrorResponse{
				Type:        ErrorTypeFileIO,
				Code:        "file_read_error",
				Title:       "File Read Error",
				Description: "Could not read the uploaded file.",
				Details:     errMsg,
				Suggestions: []string{
					"Ensure the file is not corrupted",
					"Try uploading the file again",
					"Check the file format and size",
				},
			}
		}
	}

	// Upload errors
	if strings.Contains(errMsgLower, "form") || strings.Contains(errMsgLower, "multipart") {
		return ErrorResponse{
			Type:        ErrorTypeUpload,
			Code:        "upload_form_error",
			Title:       "File Upload Error",
			Description: "There was a problem processing the uploaded file or form data.",
			Details:     errMsg,
			Suggestions: []string{
				"Check that a file was selected",
				"Ensure the file size is not too large (max 1GB)",
				"Try refreshing the page and uploading again",
			},
		}
	}

	// Default fallback for unrecognized errors
	return ErrorResponse{
		Type:        ErrorTypeInternal,
		Code:        "processing_error",
		Title:       "Processing Error",
		Description: "An error occurred while processing your request.",
		Details:     errMsg,
		Suggestions: []string{
			"Try uploading the file again",
			"Check that all form fields are filled correctly",
			"Ensure the G-code file is valid and complete",
		},
	}
}

// WriteErrorResponse writes a structured error response as JSON
func WriteErrorResponse(w http.ResponseWriter, err error, statusCode int) {
	errorResp := CategorizeError(err)
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if jsonErr := json.NewEncoder(w).Encode(errorResp); jsonErr != nil {
		// Fallback to plain text if JSON encoding fails
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "Error: %v", err)
	}
}