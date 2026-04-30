package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"disha-backend/internal/matching"
	"disha-backend/internal/ecp"
	"disha-backend/internal/students"
)

// UniversitiesHandler handles university matching and detail endpoints.
type UniversitiesHandler struct {
	matcher *matching.Matcher
	store   *students.Store
}

// NewUniversitiesHandler creates a new universities handler.
func NewUniversitiesHandler(matcher *matching.Matcher, store *students.Store) *UniversitiesHandler {
	return &UniversitiesHandler{matcher: matcher, store: store}
}

// Match handles GET /api/v1/universities/match?studentId=&country=&program=
func (h *UniversitiesHandler) Match(w http.ResponseWriter, r *http.Request) {
	studentID := r.URL.Query().Get("studentId")
	if studentID == "" {
		WriteError(w, r, "MISSING_PARAM", "studentId query parameter is required", http.StatusBadRequest)
		return
	}

	record := h.store.Get(r.Context(), studentID)
	if record == nil {
		WriteError(w, r, "STUDENT_NOT_FOUND", "No student found with ID: "+studentID, http.StatusNotFound)
		return
	}

	filters := matching.Filters{
		Country: ecp.Country(r.URL.Query().Get("country")),
		Program: ecp.ProgramType(r.URL.Query().Get("program")),
	}

	matches := h.matcher.Match(record.Profile, record.ECPResult, filters)
	WriteSuccess(w, r, matches, http.StatusOK)
}

// GetByID handles GET /api/v1/universities/{universityId}
func (h *UniversitiesHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "universityId")
	uni := h.matcher.GetByID(id)
	if uni == nil {
		WriteError(w, r, "UNIVERSITY_NOT_FOUND", "No university found with ID: "+id, http.StatusNotFound)
		return
	}
	WriteSuccess(w, r, uni, http.StatusOK)
}
