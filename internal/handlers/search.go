package handlers

import (
	"context"
	"net/http"
	"strings"

	"disha-backend/internal/data"

	"google.golang.org/api/iterator"
)

type SearchHandler struct {
	firestore *data.FirestoreClient
}

func NewSearchHandler(fc *data.FirestoreClient) *SearchHandler {
	return &SearchHandler{firestore: fc}
}

// HandleAutocomplete searches Firestore for an existing college or loan by name.
func (h *SearchHandler) HandleAutocomplete(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	query = strings.TrimSpace(query)

	if query == "" || h.firestore == nil {
		WriteSuccess(w, r, []any{}, http.StatusOK)
		return
	}

	ctx := context.Background()

	// A simple start-with prefix query against Firestore for "name"
	// For production, you'd integrate Algolia, Typesense, or use an array of tokens in Firestore.
	iter := h.firestore.Client.Collection("universities").
		Where("name", ">=", query).
		Where("name", "<=", query+"\uf8ff").
		Limit(5).
		Documents(ctx)

	var results []map[string]interface{}
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			WriteError(w, r, "FIRESTORE_ERR", "Failed to query autocomplete", http.StatusInternalServerError)
			return
		}
		data := doc.Data()
		// Only send back minimal preview structure
		results = append(results, map[string]interface{}{
			"id":      data["id"],
			"name":    data["name"],
			"country": data["country"],
			"source":  "db", // Mark as existing in database
		})
	}

	WriteSuccess(w, r, results, http.StatusOK)
}
