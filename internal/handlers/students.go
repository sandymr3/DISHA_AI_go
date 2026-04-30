package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"disha-backend/internal/ecp"
	"disha-backend/internal/students"
)

// StudentsHandler handles student CRUD endpoints.
type StudentsHandler struct {
	store *students.Store
}

// NewStudentsHandler creates a new students handler.
func NewStudentsHandler(store *students.Store) *StudentsHandler {
	return &StudentsHandler{store: store}
}

// Create handles POST /api/v1/students
func (h *StudentsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var profile ecp.StudentProfile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		WriteError(w, r, "INVALID_JSON", "Request body must be valid JSON", http.StatusBadRequest)
		return
	}

	record, err := h.store.Create(r.Context(), profile)
	if err != nil {
		WriteError(w, r, "INVALID_INPUT", err.Error(), http.StatusUnprocessableEntity)
		return
	}

	WriteSuccess(w, r, map[string]any{
		"studentId": record.StudentID,
		"ecpResult": record.ECPResult,
	}, http.StatusCreated)
}

// Get handles GET /api/v1/students/{studentId}
func (h *StudentsHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "studentId")
	record := h.store.Get(r.Context(), id)
	if record == nil {
		WriteError(w, r, "STUDENT_NOT_FOUND", "No student found with ID: "+id, http.StatusNotFound)
		return
	}
	WriteSuccess(w, r, record, http.StatusOK)
}

// Update handles PUT /api/v1/students/{studentId}
func (h *StudentsHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "studentId")
	var profile ecp.StudentProfile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		WriteError(w, r, "INVALID_JSON", "Request body must be valid JSON", http.StatusBadRequest)
		return
	}

	record, err := h.store.Update(r.Context(), id, profile)
	if err != nil {
		WriteError(w, r, "UPDATE_FAILED", err.Error(), http.StatusUnprocessableEntity)
		return
	}

	WriteSuccess(w, r, record, http.StatusOK)
}
