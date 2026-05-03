package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"disha-backend/internal/ai"
	"disha-backend/internal/data"
	"disha-backend/internal/ingestion"
)

type IngestionHandler struct {
	serper    *ingestion.SerperClient
	aiClient  *ai.Client
	firestore *data.FirestoreClient
}

func NewIngestionHandler(sc *ingestion.SerperClient, ac *ai.Client, fc *data.FirestoreClient) *IngestionHandler {
	return &IngestionHandler{
		serper:    sc,
		aiClient:  ac,
		firestore: fc,
	}
}

type IngestRequest struct {
	EntityName string `json:"entityName"`
	EntityType string `json:"entityType"` // "university" or "loan"
}

// HandleIngest coordinates searching, scraping, extracting, and saving the data.
func (h *IngestionHandler) HandleIngest(w http.ResponseWriter, r *http.Request) {
	var req IngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, "INVALID_REQUEST", "Failed to parse JSON body", http.StatusBadRequest)
		return
	}

	if h.serper == nil || h.aiClient == nil || h.firestore == nil {
		WriteError(w, r, "MISSING_INFRASTRUCTURE", "One or more ingest clients (Firebase/Serper/Gemini) are not configured", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	entityName := strings.TrimSpace(req.EntityName)
	if entityName == "" {
		WriteError(w, r, "MISSING_QUERY", "EntityName cannot be empty", http.StatusBadRequest)
		return
	}

	slog.Info("Starting dynamic ingestion pipeline", "entity", entityName, "type", req.EntityType)

	// Step 1: Serper.dev web search
	searchQuery := fmt.Sprintf("%s programs fees admissions international students site:.edu OR official", entityName)
	searchResults, err := h.serper.Search(searchQuery, 3)
	if err != nil {
		slog.Error("Serper search failed", "error", err)
		WriteError(w, r, "SEARCH_FAILED", "Failed to search the web for sources", http.StatusInternalServerError)
		return
	}

	var aggregatedText string
	for _, result := range searchResults.Organic {
		slog.Info("Scraping source", "link", result.Link)
		scraped, scrapeErr := ingestion.ScrapeURL(result.Link)
		if scrapeErr != nil {
			slog.Warn("Failed to scrape link, skipping", "link", result.Link, "err", scrapeErr)
			continue
		}
		aggregatedText += fmt.Sprintf("\n--- Source: %s ---\n%s\n", result.Link, scraped)
	}

	if len(aggregatedText) == 0 {
		WriteError(w, r, "SCRAPE_FAILED", "Could not extract any content from searched pages", http.StatusInternalServerError)
		return
	}

	// Step 2: Gemini extraction
	slog.Info("Extracting structured fields via Gemini")
	extractedUni, err := h.aiClient.ExtractUniversityData(ctx, aggregatedText)
	if err != nil {
		slog.Error("Extraction failed", "error", err)
		WriteError(w, r, "EXTRACTION_FAILED", "Failed to structure the extracted text", http.StatusInternalServerError)
		return
	}

	// Add unique ID
	id := strings.ToUpper(strings.ReplaceAll(extractedUni.Name, " ", "-"))
	if len(id) > 15 {
		id = id[:15]
	}

	// Step 3: Save to Firestore
	docRef := h.firestore.Client.Collection("universities").Doc(id)
	_, err = docRef.Set(ctx, map[string]interface{}{
		"id":                 id,
		"name":               extractedUni.Name,
		"country":            extractedUni.Country,
		"programType":        extractedUni.ProgramType,
		"programName":        extractedUni.ProgramName,
		"totalCostUSD":       extractedUni.TotalCostUSD,
		"totalCostINR":       extractedUni.TotalCostINR,
		"postStudySalaryUSD": extractedUni.PostStudySalaryUSD,
		"roiYears":           extractedUni.ROIYears,
		"admitProbability":   extractedUni.AdmitProbability,
	})

	if err != nil {
		slog.Error("Firestore save failed", "error", err)
		WriteError(w, r, "DB_SAVE_FAILED", "Failed to write document to Firestore", http.StatusInternalServerError)
		return
	}

	// Also simulate returning the matched university structure the frontend expects
	WriteSuccess(w, r, map[string]any{
		"status":  "ingested",
		"college": extractedUni,
	}, http.StatusCreated)
}
