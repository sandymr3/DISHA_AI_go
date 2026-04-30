// Package handlers provides the HTTP handlers for the DISHA API.
package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"disha-backend/internal/middleware"
)

// APIResponse is the standard JSON envelope for all API responses.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    any         `json:"data"`
	Error   *APIError   `json:"error"`
	Meta    APIMeta     `json:"meta"`
}

// APIError holds error details for client consumption.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// APIMeta holds request metadata.
type APIMeta struct {
	RequestID string `json:"requestId"`
	Timestamp string `json:"timestamp"`
}

// WriteSuccess writes a successful JSON response.
func WriteSuccess(w http.ResponseWriter, r *http.Request, data any, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    data,
		Meta:    buildMeta(r),
	})
}

// WriteError writes an error JSON response.
func WriteError(w http.ResponseWriter, r *http.Request, code string, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(APIResponse{
		Success: false,
		Error:   &APIError{Code: code, Message: message},
		Meta:    buildMeta(r),
	})
}

func buildMeta(r *http.Request) APIMeta {
	reqID, _ := r.Context().Value(middleware.RequestIDKey).(string)
	return APIMeta{
		RequestID: reqID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}
