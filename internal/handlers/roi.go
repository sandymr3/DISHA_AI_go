package handlers

import (
	"encoding/json"
	"net/http"

	"disha-backend/internal/matching"
	"disha-backend/internal/roi"
	"disha-backend/internal/students"
)

// ROIHandler handles the ROI calculation endpoint.
type ROIHandler struct {
	store   *students.Store
	matcher *matching.Matcher
}

// NewROIHandler creates a new ROI handler.
func NewROIHandler(store *students.Store, matcher *matching.Matcher) *ROIHandler {
	return &ROIHandler{store: store, matcher: matcher}
}

// Calculate handles POST /api/v1/roi/calculate
func (h *ROIHandler) Calculate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StudentID         string  `json:"studentId"`
		UniversityID      string  `json:"universityId"`
		LoanAmountLakh    float64 `json:"loanAmountLakh"`
		AnnualRatePercent float64 `json:"annualRatePercent"`
		CurrentSalaryLPA  float64 `json:"currentSalaryLPA"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, "INVALID_JSON", "Request body must be valid JSON", http.StatusBadRequest)
		return
	}

	// Validate student
	record := h.store.Get(r.Context(), req.StudentID)
	if record == nil {
		WriteError(w, r, "STUDENT_NOT_FOUND", "No student found with ID: "+req.StudentID, http.StatusNotFound)
		return
	}

	// Validate university
	uni := h.matcher.GetByID(req.UniversityID)
	if uni == nil {
		WriteError(w, r, "UNIVERSITY_NOT_FOUND", "No university found with ID: "+req.UniversityID, http.StatusNotFound)
		return
	}

	// Default loan parameters if not provided
	if req.LoanAmountLakh <= 0 {
		req.LoanAmountLakh = float64(record.ECPResult.FundingBandUpper)
	}
	if req.AnnualRatePercent <= 0 {
		req.AnnualRatePercent = 10.5
	}
	if req.CurrentSalaryLPA <= 0 {
		req.CurrentSalaryLPA = 5.0
	}

	result := roi.Calculate(roi.Params{
		TotalCostINR:       uni.TotalCostINR,
		PostStudySalaryUSD: uni.PostStudySalaryUSD,
		LoanAmountLakh:     req.LoanAmountLakh,
		AnnualRatePercent:  req.AnnualRatePercent,
		RepaymentYears:     12,
		CurrentSalaryLPA:   req.CurrentSalaryLPA,
	})

	WriteSuccess(w, r, result, http.StatusOK)
}
