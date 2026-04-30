// Package students provides a thread-safe in-memory student profile store with TTL.
package students

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"disha-backend/internal/ecp"
)

// Record holds a student profile with its computed ECP result and metadata.
type Record struct {
	StudentID string            `json:"studentId"`
	Profile   ecp.StudentProfile `json:"profile"`
	ECPResult ecp.ECPResult      `json:"ecpResult"`
	CreatedAt time.Time          `json:"createdAt"`
}

// Store is a thread-safe in-memory student store with 24-hour TTL.
type Store struct {
	mu      sync.RWMutex
	records map[string]*Record
	ttl     time.Duration
}

// NewStore creates a new student store and starts the background TTL cleanup goroutine.
// The cleanup goroutine runs until the provided context is cancelled.
func NewStore(ctx context.Context, ttl time.Duration) *Store {
	s := &Store{
		records: make(map[string]*Record),
		ttl:     ttl,
	}
	go s.cleanupLoop(ctx)
	return s
}

// Create stores a new student profile, computes the ECP score, and returns the record.
func (s *Store) Create(_ context.Context, profile ecp.StudentProfile) (*Record, error) {
	result, err := ecp.Calculate(profile)
	if err != nil {
		return nil, fmt.Errorf("students.Store.Create: %w", err)
	}

	id, err := generateStudentID()
	if err != nil {
		return nil, fmt.Errorf("students.Store.Create: generating ID: %w", err)
	}

	record := &Record{
		StudentID: id,
		Profile:   profile,
		ECPResult: result,
		CreatedAt: time.Now(),
	}

	s.mu.Lock()
	s.records[id] = record
	s.mu.Unlock()

	return record, nil
}

// CreateWithID stores a profile under a specific ID (used for demo data pre-loading).
func (s *Store) CreateWithID(_ context.Context, id string, profile ecp.StudentProfile) (*Record, error) {
	result, err := ecp.Calculate(profile)
	if err != nil {
		return nil, fmt.Errorf("students.Store.CreateWithID: %w", err)
	}

	record := &Record{
		StudentID: id,
		Profile:   profile,
		ECPResult: result,
		CreatedAt: time.Now(),
	}

	s.mu.Lock()
	s.records[id] = record
	s.mu.Unlock()

	return record, nil
}

// Get retrieves a student record by ID. Returns nil if not found or expired.
func (s *Store) Get(_ context.Context, id string) *Record {
	s.mu.RLock()
	record, ok := s.records[id]
	s.mu.RUnlock()

	if !ok {
		return nil
	}
	if time.Since(record.CreatedAt) > s.ttl {
		// Expired — clean up lazily
		s.mu.Lock()
		delete(s.records, id)
		s.mu.Unlock()
		return nil
	}
	return record
}

// Update modifies a student's profile and recalculates the ECP score.
func (s *Store) Update(_ context.Context, id string, profile ecp.StudentProfile) (*Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.records[id]
	if !ok {
		return nil, fmt.Errorf("students.Store.Update: student %q: %w", id, ecp.ErrInvalidInput)
	}

	// Merge: apply non-zero fields from the update
	merged := mergeProfile(existing.Profile, profile)

	result, err := ecp.Calculate(merged)
	if err != nil {
		return nil, fmt.Errorf("students.Store.Update: %w", err)
	}

	existing.Profile = merged
	existing.ECPResult = result

	return existing, nil
}

// Count returns the number of students currently in the store.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.records)
}

// mergeProfile applies non-zero/non-empty fields from update onto base.
func mergeProfile(base, update ecp.StudentProfile) ecp.StudentProfile {
	if update.Name != "" {
		base.Name = update.Name
	}
	if update.CGPA != 0 {
		base.CGPA = update.CGPA
	}
	// GRE 0 is a valid value (plans to take), so always apply if the field is sent
	// We use a convention: if the update profile has GREScore set, use it.
	// For partial updates, the handler will only set fields that the client sends.
	if update.GREScore != 0 || update.GREScore == 0 {
		// Always accept GRE since 0 is a valid value
		base.GREScore = update.GREScore
	}
	if update.FamilyIncome != "" {
		base.FamilyIncome = update.FamilyIncome
	}
	// HasCoApplicant is a bool — always apply from update since it's explicit
	base.HasCoApplicant = update.HasCoApplicant
	if update.CoApplicantIncome != "" {
		base.CoApplicantIncome = update.CoApplicantIncome
	}
	if update.TargetCountry != "" {
		base.TargetCountry = update.TargetCountry
	}
	if update.TargetProgram != "" {
		base.TargetProgram = update.TargetProgram
	}
	if update.Intake != "" {
		base.Intake = update.Intake
	}
	return base
}

// cleanupLoop periodically removes expired records.
func (s *Store) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.cleanup()
		}
	}
}

func (s *Store) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, record := range s.records {
		if time.Since(record.CreatedAt) > s.ttl {
			delete(s.records, id)
		}
	}
}

// generateStudentID creates a cryptographically random student ID: "STU-" + 8 hex chars.
func generateStudentID() (string, error) {
	b := make([]byte, 4) // 4 bytes = 8 hex chars
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating student ID: %w", err)
	}
	return "STU-" + hex.EncodeToString(b), nil
}
