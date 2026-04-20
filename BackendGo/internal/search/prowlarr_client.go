package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ProwlarrClient struct {
	BaseURL string
	APIKey string
	client *http.Client
}

func NewProwlarrClient(baseURL, apiKey string) *ProwlarrClient {
	return &ProwlarrClient{
		BaseURL: baseURL,
		APIKey: apiKey,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *ProwlarrClient) Search(ctx context.Context, query string) ([]NormalizedResult, error) {
	apiURL := fmt.Sprintf("%s/api/v1/search", c.BaseURL)

	params := url.Values{}
	params.Set("query", query)
	params.Set("type", "search")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Api-Key", c.APIKey)
	req.URL.RawQuery = params.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("prowlarr returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var results []ProwlarrSearchResult
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, err
	}

	return c.mapResults(results), nil
}

func (c *ProwlarrClient) mapResults(results []ProwlarrSearchResult) []NormalizedResult {
	normalized := make([]NormalizedResult, 0, len(results))
	for _, r := range results {
		n := NormalizedResult{
			Title:    r.Title,
			Guid:     r.Guid,
			Link:     r.DownloadUrl,
			Size:     r.Size,
			Seeders:  r.Seeders,
			Leechers: r.Leechers,
			Indexer:  r.Indexer,
			Provider: "prowlarr",
		}

		if r.InfoHash != "" {
			n.Hash = strings.ToLower(r.InfoHash)
		}

		if !r.PublishDate.IsZero() {
			n.PubDate = &r.PublishDate
		}

		normalized = append(normalized, n)
	}
	return normalized
}

type ProwlarrSearchResult struct {
	Title        string    `json:"title"`
	Guid         string    `json:"guid"`
	DownloadUrl  string    `json:"downloadUrl"`
	Size         int64     `json:"size"`
	Seeders      int       `json:"seeders"`
	Leechers     int       `json:"leechers"`
	Indexer      string    `json:"indexer"`
	InfoHash     string    `json:"infoHash"`
	PublishDate  time.Time `json:"publishDate"`
}