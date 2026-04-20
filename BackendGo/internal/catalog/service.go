package catalog

import (
	"context"
	"time"
)

type Service struct {
	repo   *Repository
	tmdb   *TMDBClient
	imageBaseURL string
}

func NewService(repo *Repository, tmdb *TMDBClient, imageBaseURL string) *Service {
	return &Service{
		repo:   repo,
		tmdb:   tmdb,
		imageBaseURL: imageBaseURL,
	}
}

func (s *Service) ListMovies(ctx context.Context, page, size int) ([]MovieSummary, int64, error) {
	movies, total, err := s.repo.ListMovies(ctx, page, size)
	if err != nil {
		return nil, 0, err
	}

	summaries := make([]MovieSummary, len(movies))
	for i, m := range movies {
		summaries[i] = toSummary(m)
	}
	return summaries, total, nil
}

func (s *Service) GetMovieByID(ctx context.Context, id int64) (*Movie, error) {
	return s.repo.GetMovieByID(ctx, id)
}

func (s *Service) GetMovieByTmdbID(ctx context.Context, tmdbID int) (*Movie, error) {
	return s.repo.GetMovieByTmdbID(ctx, tmdbID)
}

func (s *Service) SearchTMDB(ctx context.Context, query string) ([]TMDBMovieDetail, error) {
	if len(query) < 2 {
		return nil, nil
	}
	return s.tmdb.SearchMovies(ctx, query)
}

func (s *Service) GetGenres(ctx context.Context) ([]Genre, error) {
	return s.tmdb.GetGenres(ctx)
}

func (s *Service) DiscoverTMDB(ctx context.Context, genreID int, actor, director, year string, page int) ([]TMDBMovieDetail, error) {
	actorID := 0
	directorID := 0

	if actor != "" {
		results, err := s.tmdb.SearchPerson(ctx, actor)
		if err == nil && len(results) > 0 {
			actorID = results[0].ID
		}
	}

	if director != "" {
		results, err := s.tmdb.SearchPerson(ctx, director)
		if err == nil && len(results) > 0 {
			directorID = results[0].ID
		}
	}

	return s.tmdb.DiscoverMovies(ctx, genreID, actorID, directorID, year, page)
}

func (s *Service) ImportMovieFromTMDB(ctx context.Context, tmdbID int) (*Movie, error) {
	detail, err := s.tmdb.GetMovieDetail(ctx, tmdbID)
	if err != nil {
		return nil, err
	}

	movie := &Movie{
		TmdbID:           detail.ID,
		Title:            detail.Title,
		Overview:         detail.Overview,
		PosterPath:       s.imageURL(detail.PosterPath, "w500"),
		BackdropPath:     s.imageURL(detail.BackdropPath, "w1280"),
		ReleaseDate:      detail.ReleaseDate,
		VoteAverage:      detail.VoteAverage,
		VoteCount:         detail.VoteCount,
		Popularity:        detail.Popularity,
		OriginalLanguage: detail.OriginalLanguage,
		OriginalTitle:   detail.OriginalTitle,
		CatalogAddedAt:   time.Now(),
	}

	if err := s.repo.SaveMovie(ctx, movie); err != nil {
		return nil, err
	}
	return movie, nil
}

func (s *Service) imageURL(path, size string) string {
	if path == "" {
		return ""
	}
	return s.imageBaseURL + "/" + size + path
}

func toSummary(m Movie) MovieSummary {
	return MovieSummary{
		ID:           m.ID,
		TmdbID:       m.TmdbID,
		Title:        m.Title,
		PosterPath:   m.PosterPath,
		ReleaseDate:  m.ReleaseDate,
		VoteAverage:  m.VoteAverage,
		Popularity:   m.Popularity,
		CatalogAddedAt: formatTime(m.CatalogAddedAt),
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02T15:04:05Z")
}