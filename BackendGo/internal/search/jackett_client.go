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

type JackettClient struct {
	BaseURL   string
	APIKey   string
	Indexers string
	client    *http.Client
}

func NewJackettClient(baseURL, apiKey, indexers string) *JackettClient {
	return &JackettClient{
		BaseURL:   baseURL,
		APIKey:    apiKey,
		Indexers:  indexers,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *JackettClient) Search(ctx context.Context, query string) ([]NormalizedResult, error) {
	apiURL := fmt.Sprintf("%s/api/v2.0/indexers/all/results", c.BaseURL)

	params := url.Values{}
	params.Set("apikey", c.APIKey)
	params.Set("Query", query)
	if c.Indexers != "" && c.Indexers != "all" {
		params.Set("indexers", c.Indexers)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = params.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jackett returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result JackettResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return c.mapResults(result.Results), nil
}

func (c *JackettClient) mapResults(results []JackettItem) []NormalizedResult {
	normalized := make([]NormalizedResult, 0, len(results))
	for _, item := range results {
		n := NormalizedResult{
			Title:    item.Title,
			Guid:     item.Guid,
			Link:     item.Link,
			Size:     item.Size,
			Seeders:  item.Seeders,
			Leechers: item.Leechers,
			Indexer:  item.Indexer,
			Provider: "jackett",
		}

		if item.InfoHash != "" {
			n.Hash = strings.ToLower(item.InfoHash)
		} else {
			n.Hash = extractBTIH(item.MagnetURI)
		}

		if item.PublishDate != "" {
			if t, err := time.Parse(time.RFC1123Z, item.PublishDate); err == nil {
				n.PubDate = &t
			}
		}

		normalized = append(normalized, n)
	}
	return normalized
}

type JackettResponse struct {
	Results []JackettItem `json:"Results"`
}

type JackettItem struct {
	Title      string `json:"Title"`
	Guid       string `json:"Guid"`
	Link       string `json:"Link"`
	MagnetURI  string `json:"MagnetUri"`
	InfoHash   string `json:"InfoHash"`
	Size       int64  `json:"Size"`
	Seeders    int    `json:"Seeders"`
	Leechers   int    `json:"Leechers"`
	Indexer    string `json:"Indexer"`
	PublishDate string `json:"PublishDate"`
}

func extractBTIH(magnet string) string {
	if magnet == "" {
		return ""
	}
	for _, part := range strings.Split(magnet, "&") {
		if strings.HasPrefix(part, "xt=urn:btih:") {
			hash := strings.TrimPrefix(part, "xt=urn:btih:")
			return strings.ToLower(hash)
		}
	}
	return ""
}

var _ = extractBTIH