package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"disha-backend/internal/matching"
	"disha-backend/internal/students"
)

// PassportHandler handles the funding passport endpoint.
type PassportHandler struct {
	store   *students.Store
	matcher *matching.Matcher
}

// NewPassportHandler creates a new passport handler.
func NewPassportHandler(store *students.Store, matcher *matching.Matcher) *PassportHandler {
	return &PassportHandler{store: store, matcher: matcher}
}

// Handle handles GET /api/v1/funding-passport/{studentId}
func (h *PassportHandler) Handle(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "studentId")
	record := h.store.Get(r.Context(), id)
	if record == nil {
		WriteError(w, r, "STUDENT_NOT_FOUND", "No student found with ID: "+id, http.StatusNotFound)
		return
	}

	// Get top 2 universities
	matches := h.matcher.Match(record.Profile, record.ECPResult, matching.Filters{})
	var topTwo []map[string]any
	for i, m := range matches {
		if i >= 2 {
			break
		}
		topTwo = append(topTwo, map[string]any{
			"name":          m.Name,
			"program":       m.ProgramName,
			"fundingStatus": m.FundingStatus,
		})
	}

	// NBFC tier based on ECP tier
	nbfcTier := "Standard"
	switch record.ECPResult.Tier {
	case "Green":
		nbfcTier = "Premium"
	case "Amber":
		nbfcTier = "Standard"
	case "Red":
		nbfcTier = "Basic"
	}

	WriteSuccess(w, r, map[string]any{
		"name":               record.Profile.Name,
		"ecpScore":           record.ECPResult.Score,
		"tier":               record.ECPResult.Tier,
		"fundingBandLower":   record.ECPResult.FundingBandLower,
		"fundingBandUpper":   record.ECPResult.FundingBandUpper,
		"topTwoUniversities": topTwo,
		"nbfcTier":           nbfcTier,
		"shareUrl":           "https://disha.ai/passport/" + id,
		"generatedAt":        time.Now().UTC().Format(time.RFC3339),
	}, http.StatusOK)
}
