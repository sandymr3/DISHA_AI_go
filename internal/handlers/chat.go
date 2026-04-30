package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"disha-backend/internal/ai"
	"disha-backend/internal/matching"
	"disha-backend/internal/students"
)

// ChatHandler handles the SSE streaming chat endpoint.
type ChatHandler struct {
	client      *ai.Client
	store       *students.Store
	matcher     *matching.Matcher
	rateLimiter *ai.RateLimiter
}

// NewChatHandler creates a new chat handler.
func NewChatHandler(client *ai.Client, store *students.Store, matcher *matching.Matcher, rl *ai.RateLimiter) *ChatHandler {
	return &ChatHandler{client: client, store: store, matcher: matcher, rateLimiter: rl}
}

// Handle handles POST /api/v1/chat
func (h *ChatHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StudentID string           `json:"studentId"`
		Messages  []ai.ChatMessage `json:"messages"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, "INVALID_JSON", "Request body must be valid JSON", http.StatusBadRequest)
		return
	}

	if req.StudentID == "" || len(req.Messages) == 0 {
		WriteError(w, r, "MISSING_FIELDS", "studentId and messages are required", http.StatusBadRequest)
		return
	}

	// Rate limit check
	allowed, retryAfter := h.rateLimiter.Allow(req.StudentID)
	if !allowed {
		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
		WriteError(w, r, "RATE_LIMITED", "Too many messages. Please wait before sending more.", http.StatusTooManyRequests)
		return
	}

	// Get student profile
	record := h.store.Get(r.Context(), req.StudentID)
	if record == nil {
		WriteError(w, r, "STUDENT_NOT_FOUND", "No student found with ID: "+req.StudentID, http.StatusNotFound)
		return
	}

	// Get top universities for context
	topUnis := h.matcher.Match(record.Profile, record.ECPResult, matching.Filters{})
	if len(topUnis) > 4 {
		topUnis = topUnis[:4]
	}

	// Build system prompt
	systemPrompt := ai.BuildSystemPrompt(record.Profile, record.ECPResult, topUnis)

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Try Gemini streaming, fall back to keyword-matched response
	if h.client != nil && h.client.IsAvailable() {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		err := h.client.StreamChat(ctx, systemPrompt, req.Messages, w)
		if err == nil {
			return // Success
		}
		// Fall through to fallback
	}

	// Fallback response
	lastMsg := req.Messages[len(req.Messages)-1].Content
	fallback := ai.GetFallbackResponse(lastMsg, ai.FallbackContext{
		Profile:   record.Profile,
		ECPResult: record.ECPResult,
		TopUnis:   topUnis,
	})

	// Stream fallback as SSE
	fmt.Fprintf(w, "data: %s\n\n", mustJSON(map[string]string{"type": "delta", "text": fallback}))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	fmt.Fprintf(w, "data: %s\n\n", mustJSON(map[string]any{"type": "done", "usage": map[string]int{"inputTokens": 0, "outputTokens": 0}}))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
