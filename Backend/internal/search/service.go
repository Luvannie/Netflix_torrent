package search

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type JobStore interface {
	CreateJob(ctx context.Context, query string) (int64, error)
	GetJob(ctx context.Context, id int64) (*SearchJob, error)
	GetJobResults(ctx context.Context, id int64) ([]SearchResult, error)
	ListJobs(ctx context.Context, queryFilter string, limit, offset int) ([]SearchJob, int64, error)
	CancelJob(ctx context.Context, id int64) error
	UpdateJobStatus(ctx context.Context, id int64, status SearchJobStatus, errorMsg string) error
	SaveResults(ctx context.Context, id int64, results []SearchResult) error
}

type ProviderClient interface {
	Search(ctx context.Context, query string) ([]NormalizedResult, error)
}

type Service struct {
	repo       JobStore
	jackett    ProviderClient
	prowlarr   ProviderClient
	normalizer *Normalizer
	provider   string
}

func NewService(repo JobStore, jackett ProviderClient, prowlarr ProviderClient, provider string) *Service {
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

	if err := s.validateConfiguredProviders(ctx, id); err != nil {
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
		message := "All torrent providers failed"
		if err := s.repo.UpdateJobStatus(ctx, id, StatusFailed, message); err != nil {
			return err
		}
		return errors.New(message)
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

func (s *Service) validateConfiguredProviders(ctx context.Context, id int64) error {
	switch s.provider {
	case "jackett", "prowlarr", "both":
	default:
		message := fmt.Sprintf("unsupported torrent search provider: %s", s.provider)
		if updateErr := s.repo.UpdateJobStatus(ctx, id, StatusFailed, message); updateErr != nil {
			return updateErr
		}
		return errors.New(message)
	}

	var missing []string

	if s.provider == "jackett" || s.provider == "both" {
		if s.jackett == nil {
			missing = append(missing, "jackett")
		}
	}
	if s.provider == "prowlarr" || s.provider == "both" {
		if s.prowlarr == nil {
			missing = append(missing, "prowlarr")
		}
	}

	if len(missing) == 0 {
		return nil
	}

	message := fmt.Sprintf("torrent provider client is not configured: %s", strings.Join(missing, ", "))
	if updateErr := s.repo.UpdateJobStatus(ctx, id, StatusFailed, message); updateErr != nil {
		return updateErr
	}
	return errors.New(message)
}
