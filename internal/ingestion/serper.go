package ingestion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SerperClient wraps the serper.dev API to search for colleges and loans.
type SerperClient struct {
	APIKey string
	Client *http.Client
}

type SerperRequest struct {
	Q string `json:"q"`
}

type SerperResult struct {
	Organic []struct {
		Title   string `json:"title"`
		Link    string `json:"link"`
		Snippet string `json:"snippet"`
	} `json:"organic"`
}

func NewSerperClient(apiKey string) *SerperClient {
	return &SerperClient{
		APIKey: apiKey,
		Client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Search performs a query and returns the top N links and their snippets.
func (s *SerperClient) Search(query string, topN int) (*SerperResult, error) {
	reqBody, _ := json.Marshal(SerperRequest{Q: query})
	req, err := http.NewRequest("POST", "https://google.serper.dev/search", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("creating serper request: %v", err)
	}

	req.Header.Add("X-API-KEY", s.APIKey)
	req.Header.Add("Content-Type", "application/json")

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing serper request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("serper API error: status %d, body %s", resp.StatusCode, string(b))
	}

	var result SerperResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding serper response: %v", err)
	}

	// Trim to topN
	if len(result.Organic) > topN {
		result.Organic = result.Organic[:topN]
	}

	return &result, nil
}
