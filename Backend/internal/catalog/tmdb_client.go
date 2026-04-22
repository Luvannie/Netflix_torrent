package catalog

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

type TMDBClient struct {
	APIKey           string
	ReadAccessToken  string
	BaseURL         string
	ImageBaseURL    string
	httpClient      *http.Client
}

func NewTMDBClient(apiKey, readAccessToken, baseURL, imageBaseURL string) *TMDBClient {
	return &TMDBClient{
		APIKey:           apiKey,
		ReadAccessToken:  readAccessToken,
		BaseURL:          baseURL,
		ImageBaseURL:     imageBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *TMDBClient) SearchMovies(ctx context.Context, query string) ([]TMDBMovieDetail, error) {
	if len(query) < 2 {
		return nil, nil
	}

	params := url.Values{}
	params.Set("query", query)
	params.Set("include_adult", "false")

	var resp TMDBSearchResponse
	if err := c.get(ctx, "/search/movie", params, &resp); err != nil {
		return nil, err
	}
	return resp.Results, nil
}

func (c *TMDBClient) GetMovieDetail(ctx context.Context, tmdbID int) (*TMDBMovieDetail, error) {
	var result TMDBMovieDetail
	if err := c.get(ctx, fmt.Sprintf("/movie/%d", tmdbID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *TMDBClient) GetGenres(ctx context.Context) ([]Genre, error) {
	var resp struct {
		Genres []Genre `json:"genres"`
	}
	if err := c.get(ctx, "/genre/movie/list", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Genres, nil
}

func (c *TMDBClient) DiscoverMovies(ctx context.Context, genreID int, actorID, directorID int, year string, page int) ([]TMDBMovieDetail, error) {
	if genreID == 0 && actorID == 0 && directorID == 0 && year == "" {
		return nil, nil
	}

	params := url.Values{}
	params.Set("page", fmt.Sprintf("%d", page))

	if genreID > 0 {
		params.Set("with_genres", fmt.Sprintf("%d", genreID))
	}
	if actorID > 0 {
		params.Set("with_cast", fmt.Sprintf("%d", actorID))
	}
	if directorID > 0 {
		params.Set("with_crew", fmt.Sprintf("%d", directorID))
	}
	if year != "" {
		params.Set("primary_release_year", year)
	}

	var resp TMDBSearchResponse
	if err := c.get(ctx, "/discover/movie", params, &resp); err != nil {
		return nil, err
	}
	return resp.Results, nil
}

func (c *TMDBClient) SearchPerson(ctx context.Context, query string) ([]PersonSearchResult, error) {
	if len(query) < 2 {
		return nil, nil
	}

	params := url.Values{}
	params.Set("query", query)

	var resp struct {
		Results []PersonSearchResult `json:"results"`
	}
	if err := c.get(ctx, "/search/person", params, &resp); err != nil {
		return nil, err
	}
	return resp.Results, nil
}

type PersonSearchResult struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	KnownFor []struct {
		ID   int    `json:"id"`
		Title string `json:"title"`
	} `json:"known_for"`
}

func (c *TMDBClient) get(ctx context.Context, path string, params url.Values, result interface{}) error {
	baseURL := strings.TrimRight(c.BaseURL, "/")
	reqURL := baseURL + path
	if params != nil {
		reqURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}

	if c.ReadAccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.ReadAccessToken)
	} else if c.APIKey != "" {
		q := req.URL.Query()
		q.Set("api_key", c.APIKey)
		req.URL.RawQuery = q.Encode()
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &TMDBError{StatusCode: 503, Message: "network error: " + err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return &TMDBError{StatusCode: 502, Message: "TMDB authentication failed"}
	}
	if resp.StatusCode == 404 {
		return &TMDBError{StatusCode: 404, Message: "resource not found"}
	}
	if resp.StatusCode == 429 {
		return &TMDBError{StatusCode: 503, Message: "TMDB rate limited"}
	}
	if resp.StatusCode >= 500 {
		return &TMDBError{StatusCode: 503, Message: "TMDB upstream error"}
	}
	if resp.StatusCode >= 400 {
		return &TMDBError{StatusCode: 502, Message: "TMDB bad request"}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, result)
}

func (e *TMDBError) Error() string {
	return e.Message
}
