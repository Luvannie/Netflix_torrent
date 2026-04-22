package search

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
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

type fakeSearchRepo struct {
	job             SearchJob
	statusUpdates   []SearchJobStatus
	statusMessages  []string
	savedResults    []SearchResult
}

func (f *fakeSearchRepo) CreateJob(ctx context.Context, query string) (int64, error) {
	return 1, nil
}

func (f *fakeSearchRepo) GetJob(ctx context.Context, id int64) (*SearchJob, error) {
	return &f.job, nil
}

func (f *fakeSearchRepo) ListJobs(ctx context.Context, queryFilter string, limit, offset int) ([]SearchJob, int64, error) {
	return nil, 0, nil
}

func (f *fakeSearchRepo) CancelJob(ctx context.Context, id int64) error {
	return nil
}

func (f *fakeSearchRepo) UpdateJobStatus(ctx context.Context, id int64, status SearchJobStatus, errorMsg string) error {
	f.statusUpdates = append(f.statusUpdates, status)
	f.statusMessages = append(f.statusMessages, errorMsg)
	return nil
}

func (f *fakeSearchRepo) SaveResults(ctx context.Context, id int64, results []SearchResult) error {
	f.savedResults = append(f.savedResults, results...)
	return nil
}

func (f *fakeSearchRepo) GetJobResults(ctx context.Context, id int64) ([]SearchResult, error) {
	return nil, nil
}

type fakeProviderClient struct {
	results []NormalizedResult
	err     error
}

func (f *fakeProviderClient) Search(ctx context.Context, query string) ([]NormalizedResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.results, nil
}

func TestProcessJobFailsWhenConfiguredProviderClientIsMissing(t *testing.T) {
	repo := &fakeSearchRepo{
		job: SearchJob{ID: 7, Query: "matrix", Status: StatusRequested},
	}
	service := NewService(repo, nil, nil, "jackett")

	err := service.ProcessJob(context.Background(), 7)
	if err == nil {
		t.Fatal("ProcessJob should fail when the active provider client is not configured")
	}

	if len(repo.statusUpdates) != 2 {
		t.Fatalf("statusUpdates = %d, want 2", len(repo.statusUpdates))
	}
	if repo.statusUpdates[0] != StatusSearching {
		t.Fatalf("first status = %s, want SEARCHING", repo.statusUpdates[0])
	}
	if repo.statusUpdates[1] != StatusFailed {
		t.Fatalf("second status = %s, want FAILED", repo.statusUpdates[1])
	}
	if repo.statusMessages[1] == "" {
		t.Fatal("failed status should include a reason")
	}
}

func TestProcessJobSavesResultsWhenProviderReturnsMatches(t *testing.T) {
	now := time.Now().UTC()
	repo := &fakeSearchRepo{
		job: SearchJob{ID: 8, Query: "matrix", Status: StatusRequested},
	}
	service := NewService(repo, &fakeProviderClient{
		results: []NormalizedResult{{
			Title:    "Matrix 1999 1080p",
			Guid:     "guid-1",
			Link:     "https://example.com/download",
			Permalink: "https://example.com/torrent/1",
			Size:     1234,
			PubDate:  &now,
			Seeders:  10,
			Leechers: 2,
			Indexer:  "TestIndexer",
			Provider: "jackett",
			Hash:     "abc",
			Score:    99,
		}},
	}, nil, "jackett")

	if err := service.ProcessJob(context.Background(), 8); err != nil {
		t.Fatalf("ProcessJob returned error: %v", err)
	}

	if len(repo.savedResults) != 1 {
		t.Fatalf("savedResults = %d, want 1", len(repo.savedResults))
	}
	if got := repo.statusUpdates[len(repo.statusUpdates)-1]; got != StatusSearchReady {
		t.Fatalf("final status = %s, want SEARCH_READY", got)
	}
}

func TestProcessJobFailsWhenAllActiveProvidersReturnErrors(t *testing.T) {
	repo := &fakeSearchRepo{
		job: SearchJob{ID: 9, Query: "matrix", Status: StatusRequested},
	}
	service := NewService(repo, &fakeProviderClient{err: errors.New("jackett down")}, nil, "jackett")

	err := service.ProcessJob(context.Background(), 9)
	if err == nil {
		t.Fatal("ProcessJob should fail when the active provider returns an error and no results")
	}

	if got := repo.statusUpdates[len(repo.statusUpdates)-1]; got != StatusFailed {
		t.Fatalf("final status = %s, want FAILED", got)
	}
}
