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
	return CategorizeErrorWithLang(err, "en")
}

// CategorizeErrorWithLang analyzes an error and returns an appropriate ErrorResponse with translations
func CategorizeErrorWithLang(err error, lang string) ErrorResponse {
	if err == nil {
		return ErrorResponse{
			Type:        ErrorTypeInternal,
			Code:        "unknown_error",
			Title:       GetTranslation(lang, "error_processing_title"),
			Description: GetTranslation(lang, "error_processing_description"),
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
				Title:       GetTranslation(lang, "error_custom_template_title"),
				Description: GetTranslation(lang, "error_custom_template_description"),
				Details:     errMsg,
				Suggestions: []string{
					GetTranslation(lang, "error_custom_template_suggestion_syntax"),
					GetTranslation(lang, "error_custom_template_suggestion_sections"),
					GetTranslation(lang, "error_custom_template_suggestion_variables"),
				},
			}
		}
		return ErrorResponse{
			Type:        ErrorTypeTemplate,
			Code:        "template_parsing_error",
			Title:       GetTranslation(lang, "error_template_parsing_title"),
			Description: GetTranslation(lang, "error_template_parsing_description"),
			Details:     errMsg,
			Suggestions: []string{
				GetTranslation(lang, "error_template_parsing_suggestion_printer"),
				GetTranslation(lang, "error_template_parsing_suggestion_config"),
			},
		}
	}

	// File processing errors
	if strings.Contains(errMsgLower, "marker") || strings.Contains(errMsgLower, "position") {
		return ErrorResponse{
			Type:        ErrorTypeFileProcessing,
			Code:        "marker_not_found",
			Title:       GetTranslation(lang, "error_marker_not_found_title"),
			Description: GetTranslation(lang, "error_marker_not_found_description"),
			Details:     errMsg,
			Suggestions: []string{
				GetTranslation(lang, "error_marker_not_found_suggestion_markers"),
				GetTranslation(lang, "error_marker_not_found_suggestion_profile"),
				GetTranslation(lang, "error_marker_not_found_suggestion_compatible"),
			},
		}
	}

	if strings.Contains(errMsgLower, "print command") || strings.Contains(errMsgLower, "coordinates") {
		return ErrorResponse{
			Type:        ErrorTypeFileProcessing,
			Code:        "invalid_gcode_structure",
			Title:       GetTranslation(lang, "error_invalid_gcode_title"),
			Description: GetTranslation(lang, "error_invalid_gcode_description"),
			Details:     errMsg,
			Suggestions: []string{
				GetTranslation(lang, "error_invalid_gcode_suggestion_commands"),
				GetTranslation(lang, "error_invalid_gcode_suggestion_complete"),
				GetTranslation(lang, "error_invalid_gcode_suggestion_export"),
			},
		}
	}

	// Printer configuration errors
	if strings.Contains(errMsgLower, "printer") {
		if strings.Contains(errMsgLower, "not found") || strings.Contains(errMsgLower, "load") {
			return ErrorResponse{
				Type:        ErrorTypeConfiguration,
				Code:        "printer_not_found",
				Title:       GetTranslation(lang, "error_printer_not_found_title"),
				Description: GetTranslation(lang, "error_printer_not_found_description"),
				Details:     errMsg,
				Suggestions: []string{
					GetTranslation(lang, "error_printer_not_found_suggestion_different"),
					GetTranslation(lang, "error_printer_not_found_suggestion_custom"),
				},
			}
		}
		if strings.Contains(errMsgLower, "invalid") {
			return ErrorResponse{
				Type:        ErrorTypeValidation,
				Code:        "invalid_printer_name",
				Title:       GetTranslation(lang, "error_invalid_printer_name_title"),
				Description: GetTranslation(lang, "error_invalid_printer_name_description"),
				Details:     errMsg,
				Suggestions: []string{
					GetTranslation(lang, "error_invalid_printer_name_suggestion_format"),
					GetTranslation(lang, "error_invalid_printer_name_suggestion_dropdown"),
				},
			}
		}
	}

	// Validation errors
	if strings.Contains(errMsgLower, "iteration") || strings.Contains(errMsgLower, "positive") {
		return ErrorResponse{
			Type:        ErrorTypeValidation,
			Code:        "invalid_parameters",
			Title:       GetTranslation(lang, "error_invalid_parameters_title"),
			Description: GetTranslation(lang, "error_invalid_parameters_description"),
			Details:     errMsg,
			Suggestions: []string{
				GetTranslation(lang, "error_invalid_parameters_suggestion_positive"),
				GetTranslation(lang, "error_invalid_parameters_suggestion_ranges"),
				GetTranslation(lang, "error_invalid_parameters_suggestion_fields"),
			},
		}
	}

	// File I/O errors
	if strings.Contains(errMsgLower, "file") {
		if strings.Contains(errMsgLower, "create") || strings.Contains(errMsgLower, "write") {
			return ErrorResponse{
				Type:        ErrorTypeFileIO,
				Code:        "file_write_error",
				Title:       GetTranslation(lang, "error_file_write_title"),
				Description: GetTranslation(lang, "error_file_write_description"),
				Details:     errMsg,
				Suggestions: []string{
					GetTranslation(lang, "error_file_write_suggestion_space"),
					GetTranslation(lang, "error_file_write_suggestion_retry"),
				},
			}
		}
		if strings.Contains(errMsgLower, "open") || strings.Contains(errMsgLower, "read") {
			return ErrorResponse{
				Type:        ErrorTypeFileIO,
				Code:        "file_read_error",
				Title:       GetTranslation(lang, "error_file_read_title"),
				Description: GetTranslation(lang, "error_file_read_description"),
				Details:     errMsg,
				Suggestions: []string{
					GetTranslation(lang, "error_file_read_suggestion_corrupted"),
					GetTranslation(lang, "error_file_read_suggestion_retry"),
					GetTranslation(lang, "error_file_read_suggestion_format"),
				},
			}
		}
	}

	// Upload errors
	if strings.Contains(errMsgLower, "form") || strings.Contains(errMsgLower, "multipart") {
		return ErrorResponse{
			Type:        ErrorTypeUpload,
			Code:        "upload_form_error",
			Title:       GetTranslation(lang, "error_upload_form_title"),
			Description: GetTranslation(lang, "error_upload_form_description"),
			Details:     errMsg,
			Suggestions: []string{
				GetTranslation(lang, "error_upload_form_suggestion_selected"),
				GetTranslation(lang, "error_upload_form_suggestion_size"),
				GetTranslation(lang, "error_upload_form_suggestion_refresh"),
			},
		}
	}

	// Default fallback for unrecognized errors
	return ErrorResponse{
		Type:        ErrorTypeInternal,
		Code:        "processing_error",
		Title:       GetTranslation(lang, "error_processing_title"),
		Description: GetTranslation(lang, "error_processing_description"),
		Details:     errMsg,
		Suggestions: []string{
			GetTranslation(lang, "error_processing_suggestion_retry"),
			GetTranslation(lang, "error_processing_suggestion_fields"),
			GetTranslation(lang, "error_processing_suggestion_valid"),
		},
	}
}

// WriteErrorResponse writes a structured error response as JSON
func WriteErrorResponse(w http.ResponseWriter, err error, statusCode int) {
	WriteErrorResponseWithLang(w, err, statusCode, "en")
}

// WriteErrorResponseWithLang writes a structured error response as JSON with language support
func WriteErrorResponseWithLang(w http.ResponseWriter, err error, statusCode int, lang string) {
	errorResp := CategorizeErrorWithLang(err, lang)
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if jsonErr := json.NewEncoder(w).Encode(errorResp); jsonErr != nil {
		// Fallback to plain text if JSON encoding fails
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "Error: %v", err)
	}
}