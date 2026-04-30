package handlers

import (
	"net/http"
	"time"

	"disha-backend/internal/applications"
	"disha-backend/internal/students"
)

// HealthHandler handles the health check endpoint.
type HealthHandler struct {
	studentStore *students.Store
	appStore     *applications.Store
	startTime    time.Time
	version      string
}

// NewHealthHandler creates a new health handler.
func NewHealthHandler(ss *students.Store, as *applications.Store, version string) *HealthHandler {
	return &HealthHandler{
		studentStore: ss,
		appStore:     as,
		startTime:    time.Now(),
		version:      version,
	}
}

// Handle handles GET /api/v1/health
func (h *HealthHandler) Handle(w http.ResponseWriter, r *http.Request) {
	WriteSuccess(w, r, map[string]any{
		"status":               "ok",
		"version":              h.version,
		"uptime":               time.Since(h.startTime).String(),
		"studentsInMemory":     h.studentStore.Count(),
		"applicationsInMemory": h.appStore.Count(),
	}, http.StatusOK)
}
