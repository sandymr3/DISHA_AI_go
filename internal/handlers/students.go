package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"disha-backend/internal/ecp"
	"disha-backend/internal/middleware"
	"disha-backend/internal/students"
)

// StudentStoreInterface abstracts both the in-memory Store and FirestoreStore
// so handlers work with either backend transparently.
type StudentStoreInterface interface {
	Get(ctx interface{ Value(key any) any }, id string) *students.Record
}

// StudentsHandler handles student CRUD endpoints.
type StudentsHandler struct {
	store      *students.Store
	fsStore    *students.FirestoreStore
	useFirestore bool
}

// NewStudentsHandler creates a handler backed by the in-memory store.
func NewStudentsHandler(store *students.Store) *StudentsHandler {
	return &StudentsHandler{store: store, useFirestore: false}
}

// NewStudentsHandlerWithFirestore creates a handler that prefers Firestore,
// falling back to the in-memory store when a UID is not available.
func NewStudentsHandlerWithFirestore(store *students.Store, fsStore *students.FirestoreStore) *StudentsHandler {
	return &StudentsHandler{store: store, fsStore: fsStore, useFirestore: fsStore != nil}
}

// Create handles POST /api/v1/students
func (h *StudentsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var profile ecp.StudentProfile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		WriteError(w, r, "INVALID_JSON", "Request body must be valid JSON", http.StatusBadRequest)
		return
	}

	uid := middleware.UIDFromContext(r.Context())

	// If authenticated and Firestore is available, use UID-based persistence
	if uid != "" && h.useFirestore {
		record, err := h.fsStore.Create(r.Context(), uid, profile)
		if err != nil {
			WriteError(w, r, "INVALID_INPUT", err.Error(), http.StatusUnprocessableEntity)
			return
		}
		WriteSuccess(w, r, map[string]any{
			"studentId": record.StudentID,
			"ecpResult": record.ECPResult,
		}, http.StatusCreated)
		return
	}

	// Fallback: in-memory store (unauthenticated or Firestore unavailable)
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
	uid := middleware.UIDFromContext(r.Context())

	// If authenticated user is requesting their own profile, use UID
	lookupID := id
	if uid != "" {
		lookupID = uid
	}

	// Try Firestore first
	if h.useFirestore {
		record := h.fsStore.Get(r.Context(), lookupID)
		if record != nil {
			WriteSuccess(w, r, record, http.StatusOK)
			return
		}
	}

	// Fallback: in-memory store
	record := h.store.Get(r.Context(), lookupID)
	if record == nil {
		WriteError(w, r, "STUDENT_NOT_FOUND", "No student found with ID: "+lookupID, http.StatusNotFound)
		return
	}
	WriteSuccess(w, r, record, http.StatusOK)
}

// Update handles PUT /api/v1/students/{studentId}
func (h *StudentsHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "studentId")
	uid := middleware.UIDFromContext(r.Context())

	var profile ecp.StudentProfile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		WriteError(w, r, "INVALID_JSON", "Request body must be valid JSON", http.StatusBadRequest)
		return
	}

	// Prefer UID-based Firestore update
	updateID := id
	if uid != "" {
		updateID = uid
	}

	if uid != "" && h.useFirestore {
		record, err := h.fsStore.Update(r.Context(), updateID, profile)
		if err != nil {
			WriteError(w, r, "UPDATE_FAILED", err.Error(), http.StatusUnprocessableEntity)
			return
		}
		WriteSuccess(w, r, record, http.StatusOK)
		return
	}

	// Fallback: in-memory
	record, err := h.store.Update(r.Context(), updateID, profile)
	if err != nil {
		WriteError(w, r, "UPDATE_FAILED", err.Error(), http.StatusUnprocessableEntity)
		return
	}
	WriteSuccess(w, r, record, http.StatusOK)
}
