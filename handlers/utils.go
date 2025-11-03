package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"semantic-text-processor/errors"
	"semantic-text-processor/models"
)

// writeJSONResponse writes a JSON response with the given status code
func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
	}
}

// writeErrorResponse writes an error response with the given status code
func writeErrorResponse(w http.ResponseWriter, statusCode int, message, details string) {
	errorResp := models.APIError{
		Type:    "error",
		Code:    http.StatusText(statusCode),
		Message: message,
		Details: details,
	}
	
	writeJSONResponse(w, statusCode, errorResp)
}

// writeAppErrorResponse writes an AppError as HTTP response
func writeAppErrorResponse(w http.ResponseWriter, err error) {
	if appErr, ok := errors.AsAppError(err); ok {
		apiError := models.APIError{
			Type:    string(appErr.Type),
			Code:    appErr.Code,
			Message: appErr.Message,
			Details: appErr.Details,
		}
		
		writeJSONResponse(w, appErr.GetHTTPStatusCode(), apiError)
		
		// Log error details for debugging
		log.Printf("API Error [%s]: %s - %v", appErr.Code, appErr.Message, appErr.Cause)
		return
	}
	
	// Fallback for non-AppError
	log.Printf("Unexpected error type: %v", err)
	writeErrorResponse(w, http.StatusInternalServerError, "Internal server error", err.Error())
}

// writeWarningLog logs a warning message (for non-critical errors)
func writeWarningLog(message string, err error) {
	if err != nil {
		log.Printf("WARNING: %s: %v", message, err)
	} else {
		log.Printf("WARNING: %s", message)
	}
}

// handleError processes an error and writes appropriate HTTP response
func handleError(w http.ResponseWriter, err error, fallbackMessage string) {
	if err == nil {
		return
	}
	
	// Use the new error handling system
	writeAppErrorResponse(w, err)
}

// validateRequired checks if required fields are present
func validateRequired(fields map[string]interface{}) error {
	for fieldName, value := range fields {
		if value == nil {
			return errors.NewValidationError(
				errors.ErrCodeMissingField,
				"Required field is missing: "+fieldName,
				nil,
			)
		}
		
		// Check for empty strings
		if str, ok := value.(string); ok && str == "" {
			return errors.NewValidationError(
				errors.ErrCodeMissingField,
				"Required field is empty: "+fieldName,
				nil,
			)
		}
	}
	
	return nil
}