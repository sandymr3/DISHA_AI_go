package ingestion

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Scrape extracts visible text content from a target URL.
// Useful to gather context for the Gemini model.
func ScrapeURL(targetURL string) (string, error) {
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return "", err
	}

	// Be polite but also disguise slightly to avoid simple 403s on academic portals.
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL %s: %v", targetURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, targetURL)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML from %s: %v", targetURL, err)
	}

	// Remove scripts, styles, noscripts
	doc.Find("script, style, noscript, nav, footer").Remove()

	// Extract text and clean up whitespace
	text := doc.Find("body").Text()
	cleaned := cleanText(text)

	// Truncate to avoid exploding context windows (approx 10,000 words max)
	if len(cleaned) > 50000 {
		cleaned = cleaned[:50000] + "...\n[Content Truncated]"
	}

	return cleaned, nil
}

func cleanText(text string) string {
	lines := strings.Split(text, "\n")
	var result []string
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if t != "" {
			result = append(result, t)
		}
	}
	return strings.Join(result, "\n")
}
