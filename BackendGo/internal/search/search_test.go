package search

import (
	"encoding/json"
	"testing"
)

func TestSearchJobStatusTransitions(t *testing.T) {
	tests := []struct {
		from   SearchJobStatus
		to     SearchJobStatus
		valid  bool
	}{
		{StatusRequested, StatusSearching, true},
		{StatusRequested, StatusCancelled, true},
		{StatusRequested, StatusSearchReady, false},
		{StatusSearching, StatusSearchReady, true},
		{StatusSearching, StatusFailed, true},
		{StatusSearching, StatusCancelled, true},
		{StatusSearchReady, StatusCancelled, true},
		{StatusSearchReady, StatusSearching, false},
	}

	for _, tt := range tests {
		if got := tt.from.IsValidTransition(tt.to); got != tt.valid {
			t.Errorf("IsValidTransition(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.valid)
		}
	}
}

func TestSearchJobJSONFields(t *testing.T) {
	job := SearchJob{
		ID:           1,
		Query:        "test movie",
		Status:       StatusRequested,
		ErrorMessage: "",
	}

	data, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	expected := []string{"id", "query", "status"}
	for _, field := range expected {
		if _, ok := decoded[field]; !ok {
			t.Errorf("missing field %q", field)
		}
	}
}

func TestSearchResultStructure(t *testing.T) {
	result := SearchResult{
		ID:          1,
		SearchJobID: 10,
		Guid:        "abc123",
		Title:       "Test Movie 2024 1080p BluRay",
		Link:        "https://torrent.example.com/download",
		Permalink:   "https://torrent.example.com/torrent/123",
		Size:        1500000000,
		Seeders:     100,
		Leechers:    20,
		Indexer:     "TPB",
		Provider:    "jackett",
		Hash:        "ABC123DEF456",
		Score:       85,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	expected := []string{"id", "searchJobId", "guid", "title", "link", "permalink", "size", "seeders", "leechers", "indexer", "provider", "hash", "score"}
	for _, field := range expected {
		if _, ok := decoded[field]; !ok {
			t.Errorf("missing field %q", field)
		}
	}
}

func TestValidationErrorImplementsError(t *testing.T) {
	err := &ValidationError{Field: "query", Message: "Query must not be blank"}
	if err.Error() != "Query must not be blank" {
		t.Errorf("Error() = %q", err.Error())
	}
}

func TestNormalizedResultFields(t *testing.T) {
	result := NormalizedResult{
		Title:    "Movie Title 2024 1080p WEB-DL",
		Guid:     "guid-123",
		Link:     "https://example.com/download",
		Permalink: "https://example.com/torrent",
		Size:     2000000000,
		Seeders:  50,
		Leechers: 10,
		Indexer:  "RARBG",
		Provider: "prowlarr",
		Hash:     "HASH123",
		Score:    75,
	}

	if result.Title != "Movie Title 2024 1080p WEB-DL" {
		t.Errorf("Title = %q", result.Title)
	}
	if result.Provider != "prowlarr" {
		t.Errorf("Provider = %q", result.Provider)
	}
	if result.Score != 75 {
		t.Errorf("Score = %d", result.Score)
	}
}

func TestCreateSearchJobRequestStructure(t *testing.T) {
	jsonData := `{"query": "fight club"}`

	var req CreateSearchJobRequest
	if err := json.Unmarshal([]byte(jsonData), &req); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if req.Query != "fight club" {
		t.Errorf("Query = %q", req.Query)
	}
}
