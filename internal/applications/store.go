// Package applications provides a thread-safe in-memory loan application store.
package applications

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"
)

// Status represents the current state of a loan application.
type Status string

const (
	StatusSubmitted         Status = "Submitted"
	StatusUnderReview       Status = "Under Review"
	StatusDocumentsVerified Status = "Documents Verified"
	StatusSanctioned        Status = "Sanctioned"
)

// DocumentItem represents one item in the document checklist.
type DocumentItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Submitted   bool   `json:"submitted"`
}

// Application represents a loan application record.
type Application struct {
	ApplicationID      string         `json:"applicationId"`
	StudentID          string         `json:"studentId"`
	LoanOfferID        string         `json:"loanOfferId"`
	RequestedAmountLakh float64       `json:"requestedAmountLakh"`
	Phone              string         `json:"phone"`
	PANLastFour        string         `json:"panLastFour"`
	TargetUniversityID string         `json:"targetUniversityId"`
	Status             Status         `json:"status"`
	EstimatedDays      int            `json:"estimatedDays"`
	DocumentChecklist  []DocumentItem `json:"documentChecklist"`
	CreatedAt          time.Time      `json:"createdAt"`
}

// CreateRequest is the input for creating a new application.
type CreateRequest struct {
	StudentID          string  `json:"studentId"`
	LoanOfferID        string  `json:"loanOfferId"`
	RequestedAmountLakh float64 `json:"requestedAmountLakh"`
	Phone              string  `json:"phone"`
	PANLastFour        string  `json:"panLastFour"`
	TargetUniversityID string  `json:"targetUniversityId"`
}

// Store is a thread-safe in-memory application store.
type Store struct {
	mu           sync.RWMutex
	applications map[string]*Application
}

// NewStore creates a new application store.
func NewStore() *Store {
	return &Store{
		applications: make(map[string]*Application),
	}
}

// Create stores a new loan application and returns the created record.
func (s *Store) Create(_ context.Context, req CreateRequest) (*Application, error) {
	appID, err := generateApplicationID()
	if err != nil {
		return nil, fmt.Errorf("applications.Store.Create: %w", err)
	}

	app := &Application{
		ApplicationID:      appID,
		StudentID:          req.StudentID,
		LoanOfferID:        req.LoanOfferID,
		RequestedAmountLakh: req.RequestedAmountLakh,
		Phone:              req.Phone,
		PANLastFour:        req.PANLastFour,
		TargetUniversityID: req.TargetUniversityID,
		Status:             StatusSubmitted,
		EstimatedDays:      7,
		DocumentChecklist:  defaultChecklist(),
		CreatedAt:          time.Now(),
	}

	s.mu.Lock()
	s.applications[appID] = app
	s.mu.Unlock()

	return app, nil
}

// Get retrieves an application by ID.
func (s *Store) Get(_ context.Context, id string) *Application {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.applications[id]
}

// Count returns the number of applications in the store.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.applications)
}

// defaultChecklist returns the standard document checklist for loan applications.
func defaultChecklist() []DocumentItem {
	return []DocumentItem{
		{Name: "Income Proof", Description: "ITR or salary slips for last 2 years", Required: true, Submitted: false},
		{Name: "Address Proof", Description: "Aadhaar card or Passport", Required: true, Submitted: false},
		{Name: "Academic Transcripts", Description: "All semester marksheets and degree certificate", Required: true, Submitted: false},
		{Name: "Offer Letter", Description: "Conditional or unconditional offer from university", Required: false, Submitted: false},
	}
}

// generateApplicationID creates: "DISHA-2025-" + 6 random digits.
func generateApplicationID() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", fmt.Errorf("generating application ID: %w", err)
	}
	return fmt.Sprintf("DISHA-2025-%06d", n.Int64()), nil
}
