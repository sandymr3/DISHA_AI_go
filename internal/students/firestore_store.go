// Package students provides Firestore-backed student profile persistence.
package students

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"cloud.google.com/go/firestore"
	"disha-backend/internal/data"
	"disha-backend/internal/ecp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const studentsCollection = "students"

// firestoreRecord is the on-disk representation stored in Firestore.
type firestoreRecord struct {
	StudentID string            `firestore:"studentId"`
	Profile   ecp.StudentProfile `firestore:"profile"`
	ECPResult ecp.ECPResult     `firestore:"ecpResult"`
	CreatedAt time.Time         `firestore:"createdAt"`
	UpdatedAt time.Time         `firestore:"updatedAt"`
}

// FirestoreStore implements persistent student storage backed by Firestore.
// The Firebase UID is used as the Firestore document ID.
type FirestoreStore struct {
	db *data.FirestoreClient
}

// NewFirestoreStore creates a new FirestoreStore wrapping the provided client.
func NewFirestoreStore(db *data.FirestoreClient) *FirestoreStore {
	return &FirestoreStore{db: db}
}

// Create persists a new student profile using uid as the document ID.
// If a document with that uid already exists, it is overwritten.
func (s *FirestoreStore) Create(ctx context.Context, uid string, profile ecp.StudentProfile) (*Record, error) {
	result, err := ecp.Calculate(profile)
	if err != nil {
		return nil, fmt.Errorf("FirestoreStore.Create: ecp calculate: %w", err)
	}

	now := time.Now()
	doc := firestoreRecord{
		StudentID: uid,
		Profile:   profile,
		ECPResult: result,
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err = s.db.Client.Collection(studentsCollection).Doc(uid).Set(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("FirestoreStore.Create: firestore set: %w", err)
	}

	slog.Info("student profile created in Firestore", "uid", uid, "ecp_score", result.Score)

	return &Record{
		StudentID: uid,
		Profile:   profile,
		ECPResult: result,
		CreatedAt: now,
	}, nil
}

// Get retrieves a student record by uid. Returns nil if not found.
func (s *FirestoreStore) Get(ctx context.Context, uid string) *Record {
	snap, err := s.db.Client.Collection(studentsCollection).Doc(uid).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil
		}
		slog.Error("FirestoreStore.Get: firestore error", "uid", uid, "error", err)
		return nil
	}

	var doc firestoreRecord
	if err := snap.DataTo(&doc); err != nil {
		slog.Error("FirestoreStore.Get: data unmarshal error", "uid", uid, "error", err)
		return nil
	}

	return &Record{
		StudentID: doc.StudentID,
		Profile:   doc.Profile,
		ECPResult: doc.ECPResult,
		CreatedAt: doc.CreatedAt,
	}
}

// Update modifies an existing student profile and recalculates ECP.
func (s *FirestoreStore) Update(ctx context.Context, uid string, profile ecp.StudentProfile) (*Record, error) {
	// Read existing
	existing := s.Get(ctx, uid)
	if existing == nil {
		// If no existing, create fresh
		return s.Create(ctx, uid, profile)
	}

	// Merge and recalculate
	merged := mergeProfile(existing.Profile, profile)
	result, err := ecp.Calculate(merged)
	if err != nil {
		return nil, fmt.Errorf("FirestoreStore.Update: ecp calculate: %w", err)
	}

	updates := []firestore.Update{
		{Path: "profile", Value: merged},
		{Path: "ecpResult", Value: result},
		{Path: "updatedAt", Value: time.Now()},
	}

	_, err = s.db.Client.Collection(studentsCollection).Doc(uid).Update(ctx, updates)
	if err != nil {
		return nil, fmt.Errorf("FirestoreStore.Update: firestore update: %w", err)
	}

	slog.Info("student profile updated in Firestore", "uid", uid, "ecp_score", result.Score)

	return &Record{
		StudentID: uid,
		Profile:   merged,
		ECPResult: result,
		CreatedAt: existing.CreatedAt,
	}, nil
}
