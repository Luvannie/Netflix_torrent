package search

import (
	"context"
	"strings"
)

type Service struct {
	repo       *Repository
	jackett    *JackettClient
	prowlarr   *ProwlarrClient
	normalizer *Normalizer
	provider   string
}

func NewService(repo *Repository, jackett *JackettClient, prowlarr *ProwlarrClient, provider string) *Service {
	return &Service{
		repo:       repo,
		jackett:    jackett,
		prowlarr:   prowlarr,
		normalizer: NewNormalizer(),
		provider:   provider,
	}
}

func (s *Service) CreateJob(ctx context.Context, query string) (int64, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return 0, &ValidationError{Field: "query", Message: "Query must not be blank"}
	}
	return s.repo.CreateJob(ctx, query)
}

func (s *Service) GetJob(ctx context.Context, id int64) (*SearchJob, error) {
	job, err := s.repo.GetJob(ctx, id)
	if err != nil {
		return nil, err
	}
	results, err := s.repo.GetJobResults(ctx, id)
	if err == nil {
		job.Results = results
	}
	return job, nil
}

func (s *Service) ListJobs(ctx context.Context, queryFilter string, limit, offset int) ([]SearchJob, int64, error) {
	return s.repo.ListJobs(ctx, queryFilter, limit, offset)
}

func (s *Service) CancelJob(ctx context.Context, id int64) error {
	return s.repo.CancelJob(ctx, id)
}

func (s *Service) ProcessJob(ctx context.Context, id int64) error {
	job, err := s.repo.GetJob(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repo.UpdateJobStatus(ctx, id, StatusSearching, ""); err != nil {
		return err
	}

	var allResults []ProviderResult

	if s.provider == "jackett" || s.provider == "both" {
		if s.jackett != nil {
			results, err := s.jackett.Search(ctx, job.Query)
			allResults = append(allResults, ProviderResult{
				Provider: "jackett",
				Results:  results,
				Error:   err,
			})
		}
	}

	if s.provider == "prowlarr" || s.provider == "both" {
		if s.prowlarr != nil {
			results, err := s.prowlarr.Search(ctx, job.Query)
			allResults = append(allResults, ProviderResult{
				Provider: "prowlarr",
				Results:  results,
				Error:   err,
			})
		}
	}

	normalized := s.normalizer.Normalize(allResults)

	hasError := false
	for _, r := range allResults {
		if r.Error != nil {
			hasError = true
			break
		}
	}

	if len(normalized) == 0 && hasError {
		return s.repo.UpdateJobStatus(ctx, id, StatusFailed, "All torrent providers failed")
	}

	var searchResults []SearchResult
	for _, n := range normalized {
		searchResults = append(searchResults, SearchResult{
			SearchJobID: id,
			Guid:       n.Guid,
			Title:      n.Title,
			Link:       n.Link,
			Permalink:  n.Permalink,
			Size:       n.Size,
			PubDate:    n.PubDate,
			Seeders:    n.Seeders,
			Leechers:   n.Leechers,
			Indexer:    n.Indexer,
			Provider:   n.Provider,
			Hash:       n.Hash,
			Score:      n.Score,
		})
	}

	if err := s.repo.SaveResults(ctx, id, searchResults); err != nil {
		return err
	}

	return s.repo.UpdateJobStatus(ctx, id, StatusSearchReady, "")
}

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}