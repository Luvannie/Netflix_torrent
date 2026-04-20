package catalog

import "time"

type Movie struct {
	ID              int64      `json:"id"`
	TmdbID         int        `json:"tmdbId"`
	Title          string     `json:"title"`
	Overview       string     `json:"overview"`
	PosterPath     string     `json:"posterPath"`
	BackdropPath   string     `json:"backdropPath"`
	ReleaseDate    string     `json:"releaseDate"`
	VoteAverage    float64    `json:"voteAverage"`
	VoteCount      int        `json:"voteCount"`
	Popularity     float64    `json:"popularity"`
	OriginalLanguage string   `json:"originalLanguage"`
	OriginalTitle  string     `json:"originalTitle"`
	CatalogAddedAt time.Time  `json:"catalogAddedAt"`
}

type MovieSummary struct {
	ID              int64   `json:"id"`
	TmdbID         int      `json:"tmdbId"`
	Title          string   `json:"title"`
	PosterPath     string   `json:"posterPath"`
	ReleaseDate    string   `json:"releaseDate"`
	VoteAverage    float64  `json:"voteAverage"`
	Popularity     float64  `json:"popularity"`
	CatalogAddedAt string   `json:"catalogAddedAt"`
}

type TMDBMovieDetail struct {
	ID               int     `json:"id"`
	Title            string  `json:"title"`
	Overview         string  `json:"overview"`
	PosterPath       string  `json:"poster_path"`
	BackdropPath     string  `json:"backdrop_path"`
	ReleaseDate       string  `json:"release_date"`
	VoteAverage      float64 `json:"vote_average"`
	VoteCount        int     `json:"vote_count"`
	Popularity       float64 `json:"popularity"`
	OriginalLanguage string  `json:"original_language"`
	OriginalTitle    string  `json:"original_title"`
	Genres           []Genre `json:"genres"`
}

type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type TMDBSearchResponse struct {
	Results []TMDBMovieDetail `json:"results"`
	TotalPages int            `json:"total_pages"`
}

type TMDBError struct {
	StatusCode int
	Message    string
}