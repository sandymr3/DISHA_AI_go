package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"google.golang.org/genai"
)

// Client wraps the Google GenAI SDK for streaming chat responses.
type Client struct {
	client *genai.Client
	model  string
}

// ChatMessage represents a single message in the conversation.
type ChatMessage struct {
	Role    string `json:"role"`    // "user" or "assistant" (mapped to "model" for Gemini)
	Content string `json:"content"`
}

// NewClient creates a new Gemini AI client. Returns nil if apiKey is empty.
func NewClient(ctx context.Context, apiKey string) (*Client, error) {
	if apiKey == "" {
		return nil, nil
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("ai.NewClient: %w", err)
	}

	return &Client{
		client: client,
		model:  "gemini-2.5-flash",
	}, nil
}

// IsAvailable returns true if the client is configured and ready.
func (c *Client) IsAvailable() bool {
	return c != nil && c.client != nil
}

// StreamChat streams a response from Gemini to the HTTP ResponseWriter as SSE events.
func (c *Client) StreamChat(ctx context.Context, systemPrompt string, messages []ChatMessage, w http.ResponseWriter) error {
	// Build the contents from conversation history
	var contents []*genai.Content
	for _, msg := range messages {
		role := msg.Role
		if role == "assistant" {
			role = "model"
		}
		contents = append(contents, &genai.Content{
			Role: role,
			Parts: []*genai.Part{
				genai.NewPartFromText(msg.Content),
			},
		})
	}

	config := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{
				genai.NewPartFromText(systemPrompt),
			},
		},
		MaxOutputTokens: 400,
	}

	stream := c.client.Models.GenerateContentStream(ctx, c.model, contents, config)

	var totalInputTokens, totalOutputTokens int32

	for resp, err := range stream {
		if err != nil {
			slog.Error("gemini stream error", "error", err)
			writeSSEEvent(w, "error", map[string]string{"message": "AI service temporarily unavailable"})
			return fmt.Errorf("ai.StreamChat: stream error: %w", err)
		}

		// Extract text from candidates
		if resp.Candidates != nil {
			for _, cand := range resp.Candidates {
				if cand.Content != nil {
					for _, part := range cand.Content.Parts {
						if part.Text != "" {
							writeSSEEvent(w, "delta", map[string]string{"text": part.Text})
						}
					}
				}
			}
		}

		// Track token usage
		if resp.UsageMetadata != nil {
			totalInputTokens = resp.UsageMetadata.PromptTokenCount
			totalOutputTokens = resp.UsageMetadata.CandidatesTokenCount
		}
	}

	// Send done event with usage
	writeSSEEvent(w, "done", map[string]any{
		"usage": map[string]int32{
			"inputTokens":  totalInputTokens,
			"outputTokens": totalOutputTokens,
		},
	})

	return nil
}

// writeSSEEvent writes a single SSE event to the ResponseWriter and flushes.
func writeSSEEvent(w http.ResponseWriter, eventType string, data any) {
	payload := map[string]any{
		"type": eventType,
	}

	// Merge data fields into payload
	switch d := data.(type) {
	case map[string]string:
		for k, v := range d {
			payload[k] = v
		}
	case map[string]any:
		for k, v := range d {
			payload[k] = v
		}
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		slog.Error("SSE marshal error", "error", err)
		return
	}

	fmt.Fprintf(w, "data: %s\n\n", jsonBytes)

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}
