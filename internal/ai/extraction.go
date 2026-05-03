package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/genai"
)

// ExtractedUniversity is the target JSON schema we want Gemini to fill out.
type ExtractedUniversity struct {
	Name               string             `json:"name"`
	Country            string             `json:"country"`
	ProgramType        string             `json:"programType"`
	ProgramName        string             `json:"programName"`
	TotalCostUSD       int                `json:"totalCostUSD"`
	TotalCostINR       int64              `json:"totalCostINR"`
	PostStudySalaryUSD int                `json:"postStudySalaryUSD"`
	ROIYears           float64            `json:"roiYears"`
	AdmitProbability   map[string]float64 `json:"admitProbability"`
}

// ExtractUniversityData takes raw scraped text and uses Gemini to map it to a structured schema.
func (c *Client) ExtractUniversityData(ctx context.Context, contextText string) (*ExtractedUniversity, error) {
	systemPrompt := `You are an expert education data extractor. 
Extract university program data from the provided raw website text.
Always output valid JSON satisfying the schema.
If a fee is provided in a foreign currency, estimate it in INR (assuming 1 USD = 83 INR, 1 GBP = 105 INR) and provide both.
If exactly one program is described, extract it. If multiple are described, pick the most prominent target program (like MS in CS or MBA).
Make an educated estimate for postStudySalaryUSD as a single integer value.
Make an educated estimate for admitProbability as a dictionary mapped to "Green", "Amber", "Red" student tiers based on the rank.
Return ONLY the raw JSON string.`

	config := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{genai.NewPartFromText(systemPrompt)},
		},
		Temperature: genai.Ptr[float32](0.1),
		// Note: The newer SDK might use ResponseMimeType "application/json"
		ResponseMIMEType: "application/json",
	}

	contents := []*genai.Content{
		{
			Role: "user",
			Parts: []*genai.Part{
				genai.NewPartFromText(fmt.Sprintf("RAW TEXT:\n\n%s", contextText)),
			},
		},
	}

	resp, err := c.client.Models.GenerateContent(ctx, c.model, contents, config)
	if err != nil {
		return nil, fmt.Errorf("gemini extraction error: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini returned no content")
	}

	// Assuming part is text
	responseText := resp.Candidates[0].Content.Parts[0].Text
	
	// Sometimes there's markdown ```json ... ``` wrapper.
	// Clean it up just in case, though ResponseMimeType = application/json usually prevents it.

	var uni ExtractedUniversity
	if err := json.Unmarshal([]byte(responseText), &uni); err != nil {
		return nil, fmt.Errorf("failed to parse extracted JSON: %w (raw: %s)", err, responseText)
	}

	return &uni, nil
}
