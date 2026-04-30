package handlers

import (
	"fmt"
	"math"
	"net/http"

	"disha-backend/internal/ecp"
	"disha-backend/internal/matching"
	"disha-backend/internal/students"
)

// DreamGapHandler handles the dream gap analysis endpoint.
type DreamGapHandler struct {
	store   *students.Store
	matcher *matching.Matcher
}

// NewDreamGapHandler creates a new dream gap handler.
func NewDreamGapHandler(store *students.Store, matcher *matching.Matcher) *DreamGapHandler {
	return &DreamGapHandler{store: store, matcher: matcher}
}

// GapPath represents one way to close the funding gap.
type GapPath struct {
	Action             string  `json:"action"`
	PotentialUnlockLakh float64 `json:"potentialUnlockLakh"`
	Effort             string  `json:"effort"`
}

// Handle handles GET /api/v1/dream-gap?universityId=&studentId=
func (h *DreamGapHandler) Handle(w http.ResponseWriter, r *http.Request) {
	studentID := r.URL.Query().Get("studentId")
	universityID := r.URL.Query().Get("universityId")

	if studentID == "" || universityID == "" {
		WriteError(w, r, "MISSING_PARAMS", "studentId and universityId are required", http.StatusBadRequest)
		return
	}

	record := h.store.Get(r.Context(), studentID)
	if record == nil {
		WriteError(w, r, "STUDENT_NOT_FOUND", "No student found with ID: "+studentID, http.StatusNotFound)
		return
	}

	uni := h.matcher.GetByID(universityID)
	if uni == nil {
		WriteError(w, r, "UNIVERSITY_NOT_FOUND", "No university found with ID: "+universityID, http.StatusNotFound)
		return
	}

	costLakh := float64(uni.TotalCostINR) / 100.0 / 100000.0
	fundingUpper := float64(record.ECPResult.FundingBandUpper)
	gap := math.Max(0, costLakh-fundingUpper)

	var paths []GapPath
	if !record.Profile.HasCoApplicant {
		paths = append(paths, GapPath{
			Action:             "Add a co-applicant with income above ₹8L/year",
			PotentialUnlockLakh: 10,
			Effort:             "Medium",
		})
	}
	paths = append(paths, GapPath{
		Action:             "Apply for merit-based scholarships at " + uni.Name,
		PotentialUnlockLakh: math.Round(costLakh * 0.15),
		Effort:             "High",
	})

	// Find 2 alternative cheaper universities in the same country
	alternatives := h.findAlternatives(uni.Country, uni.ProgramType, uni.TotalCostINR, record)
	for _, alt := range alternatives {
		altCostLakh := float64(alt.TotalCostINR) / 100.0 / 100000.0
		saving := costLakh - altCostLakh
		paths = append(paths, GapPath{
			Action:             "Consider " + alt.Name + " (" + alt.ProgramName + ") — saves ₹" + formatLakh(saving) + "L",
			PotentialUnlockLakh: math.Round(saving),
			Effort:             "Low",
		})
	}

	WriteSuccess(w, r, map[string]any{
		"dreamUniversityName":       uni.Name,
		"dreamUniversityTotalCostLakh": math.Round(costLakh),
		"studentFundingUpperLakh":   fundingUpper,
		"gapLakh":                   math.Round(gap),
		"pathsToCloseGap":           paths,
	}, http.StatusOK)
}

func (h *DreamGapHandler) findAlternatives(country ecp.Country, program ecp.ProgramType, currentCostINR int64, record *students.Record) []matching.University {
	allUnis := h.matcher.AllUniversities()
	var cheaper []matching.University
	for _, u := range allUnis {
		if u.Country == country && u.ProgramType == program && u.TotalCostINR < currentCostINR {
			cheaper = append(cheaper, u)
		}
	}
	if len(cheaper) > 2 {
		cheaper = cheaper[:2]
	}
	return cheaper
}

func formatLakh(v float64) string {
	if v == float64(int(v)) {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.1f", v)
}
