package handlers

import (
	"encoding/json"
	"net/http"

	"disha-backend/internal/ecp"
)

// ECPHandler handles ECP scoring endpoints.
type ECPHandler struct{}

// NewECPHandler creates a new ECP handler.
func NewECPHandler() *ECPHandler { return &ECPHandler{} }

// Calculate handles POST /api/v1/ecp/calculate
func (h *ECPHandler) Calculate(w http.ResponseWriter, r *http.Request) {
	var profile ecp.StudentProfile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		WriteError(w, r, "INVALID_JSON", "Request body must be valid JSON", http.StatusBadRequest)
		return
	}

	result, err := ecp.Calculate(profile)
	if err != nil {
		WriteError(w, r, "INVALID_INPUT", err.Error(), http.StatusUnprocessableEntity)
		return
	}

	WriteSuccess(w, r, result, http.StatusOK)
}

// Simulate handles POST /api/v1/ecp/simulate — same as Calculate but explicitly stateless.
func (h *ECPHandler) Simulate(w http.ResponseWriter, r *http.Request) {
	h.Calculate(w, r)
}
