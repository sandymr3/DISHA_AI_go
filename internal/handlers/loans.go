package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"disha-backend/internal/applications"
	"disha-backend/internal/loans"
	"disha-backend/internal/students"
)

// LoansHandler handles loan offer, EMI, and application endpoints.
type LoansHandler struct {
	offers   []loans.LoanOffer
	store    *students.Store
	appStore *applications.Store
}

// NewLoansHandler creates a new loans handler.
func NewLoansHandler(offers []loans.LoanOffer, store *students.Store, appStore *applications.Store) *LoansHandler {
	return &LoansHandler{offers: offers, store: store, appStore: appStore}
}

// GetOffers handles GET /api/v1/loans/offers?studentId=
func (h *LoansHandler) GetOffers(w http.ResponseWriter, r *http.Request) {
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

	result := loans.FilterAndRankOffers(
		record.ECPResult.Score,
		record.ECPResult.FundingBandLower,
		record.ECPResult.FundingBandUpper,
		h.offers,
	)
	WriteSuccess(w, r, result, http.StatusOK)
}

// CalculateEMI handles POST /api/v1/loans/emi
func (h *LoansHandler) CalculateEMI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		LoanAmountLakh    float64 `json:"loanAmountLakh"`
		AnnualRatePercent float64 `json:"annualRatePercent"`
		RepaymentYears    int     `json:"repaymentYears"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, "INVALID_JSON", "Request body must be valid JSON", http.StatusBadRequest)
		return
	}

	if req.LoanAmountLakh < 1 || req.LoanAmountLakh > 100 {
		WriteError(w, r, "INVALID_AMOUNT", "Loan amount must be between 1 and 100 Lakhs", http.StatusUnprocessableEntity)
		return
	}
	if req.AnnualRatePercent < 5 || req.AnnualRatePercent > 25 {
		WriteError(w, r, "INVALID_RATE", "Interest rate must be between 5% and 25%", http.StatusUnprocessableEntity)
		return
	}
	if req.RepaymentYears < 1 || req.RepaymentYears > 20 {
		WriteError(w, r, "INVALID_TENURE", "Repayment years must be between 1 and 20", http.StatusUnprocessableEntity)
		return
	}

	result := loans.CalculateEMI(req.LoanAmountLakh, req.AnnualRatePercent, req.RepaymentYears)
	WriteSuccess(w, r, result, http.StatusOK)
}

// CreateApplication handles POST /api/v1/applications
func (h *LoansHandler) CreateApplication(w http.ResponseWriter, r *http.Request) {
	var req applications.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, "INVALID_JSON", "Request body must be valid JSON", http.StatusBadRequest)
		return
	}

	if req.StudentID == "" || req.LoanOfferID == "" {
		WriteError(w, r, "MISSING_FIELDS", "studentId and loanOfferId are required", http.StatusBadRequest)
		return
	}

	app, err := h.appStore.Create(r.Context(), req)
	if err != nil {
		WriteError(w, r, "APPLICATION_ERROR", err.Error(), http.StatusInternalServerError)
		return
	}

	WriteSuccess(w, r, app, http.StatusCreated)
}

// GetApplication handles GET /api/v1/applications/{applicationId}
func (h *LoansHandler) GetApplication(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "applicationId")
	app := h.appStore.Get(r.Context(), id)
	if app == nil {
		WriteError(w, r, "APPLICATION_NOT_FOUND", "No application found with ID: "+id, http.StatusNotFound)
		return
	}
	WriteSuccess(w, r, app, http.StatusOK)
}
