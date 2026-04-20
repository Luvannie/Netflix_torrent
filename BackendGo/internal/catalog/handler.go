package catalog

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/netflixtorrent/backend-go/internal/api"
	"github.com/netflixtorrent/backend-go/internal/httpx"
	"github.com/netflixtorrent/backend-go/internal/pagination"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Routes() map[string]http.Handler {
	return map[string]http.Handler{
		"GET /api/v1/catalog":                   http.HandlerFunc(h.listCatalog),
		"GET /api/v1/catalog/{id}":              http.HandlerFunc(h.getByID),
		"GET /api/v1/catalog/movies/{tmdbId}":     http.HandlerFunc(h.getByTmdbID),
		"GET /api/v1/catalog/search":              http.HandlerFunc(h.search),
		"GET /api/v1/catalog/genres":              http.HandlerFunc(h.genres),
		"GET /api/v1/catalog/discover":            http.HandlerFunc(h.discover),
	}
}

func (h *Handler) listCatalog(w http.ResponseWriter, r *http.Request) {
	page, size := pagination.Parse(r)
	movies, total, err := h.service.ListMovies(r.Context(), page, size)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Failed to list catalog", nil, httpx.InboundRequestID(r))
		return
	}

	if movies == nil {
		movies = []MovieSummary{}
	}

	api.WriteOK(w, http.StatusOK, pagination.New(movies, page, size, total), httpx.InboundRequestID(r))
}

func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathID(r.URL.Path, 3)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid movie ID", nil, httpx.InboundRequestID(r))
		return
	}

	movie, err := h.service.GetMovieByID(r.Context(), id)
	if err != nil {
		api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Movie not found", nil, httpx.InboundRequestID(r))
		return
	}

	api.WriteOK(w, http.StatusOK, movie, httpx.InboundRequestID(r))
}

func (h *Handler) getByTmdbID(w http.ResponseWriter, r *http.Request) {
	tmdbID, err := parsePathID(r.URL.Path, 4)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid TMDB ID", nil, httpx.InboundRequestID(r))
		return
	}

	movie, err := h.service.GetMovieByTmdbID(r.Context(), int(tmdbID))
	if err != nil {
		api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Movie not found", nil, httpx.InboundRequestID(r))
		return
	}

	api.WriteOK(w, http.StatusOK, movie, httpx.InboundRequestID(r))
}

func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if query == "" || len(query) < 2 {
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Query must be at least 2 characters", nil, httpx.InboundRequestID(r))
		return
	}

	results, err := h.service.SearchTMDB(r.Context(), query)
	if err != nil {
		if tmdbErr, ok := err.(*TMDBError); ok {
			if tmdbErr.StatusCode == 502 {
				api.WriteError(w, http.StatusBadGateway, "TMDB_AUTH_ERROR", tmdbErr.Message, nil, httpx.InboundRequestID(r))
				return
			}
			if tmdbErr.StatusCode == 503 {
				api.WriteError(w, http.StatusServiceUnavailable, "TMDB_RATE_LIMITED", tmdbErr.Message, nil, httpx.InboundRequestID(r))
				return
			}
		}
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "TMDB search failed", nil, httpx.InboundRequestID(r))
		return
	}

	if results == nil {
		results = []TMDBMovieDetail{}
	}

	api.WriteOK(w, http.StatusOK, results, httpx.InboundRequestID(r))
}

func (h *Handler) genres(w http.ResponseWriter, r *http.Request) {
	genres, err := h.service.GetGenres(r.Context())
	if err != nil {
		if tmdbErr, ok := err.(*TMDBError); ok {
			api.WriteError(w, http.StatusBadGateway, "TMDB_ERROR", tmdbErr.Message, nil, httpx.InboundRequestID(r))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Failed to get genres", nil, httpx.InboundRequestID(r))
		return
	}

	if genres == nil {
		genres = []Genre{}
	}

	api.WriteOK(w, http.StatusOK, genres, httpx.InboundRequestID(r))
}

func (h *Handler) discover(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	genreID := intParam(query.Get("genreId"))
	actor := strings.TrimSpace(query.Get("actor"))
	director := strings.TrimSpace(query.Get("director"))
	year := strings.TrimSpace(query.Get("year"))
	page := intParam(query.Get("page"))
	if page < 1 {
		page = 1
	}

	if genreID == 0 && actor == "" && director == "" && year == "" {
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "At least one filter is required", nil, httpx.InboundRequestID(r))
		return
	}

	if actor != "" && len(actor) < 2 {
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Actor name must be at least 2 characters", nil, httpx.InboundRequestID(r))
		return
	}

	if director != "" && len(director) < 2 {
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Director name must be at least 2 characters", nil, httpx.InboundRequestID(r))
		return
	}

	results, err := h.service.DiscoverTMDB(r.Context(), genreID, actor, director, year, page)
	if err != nil {
		if tmdbErr, ok := err.(*TMDBError); ok {
			api.WriteError(w, http.StatusBadGateway, "TMDB_ERROR", tmdbErr.Message, nil, httpx.InboundRequestID(r))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Discover failed", nil, httpx.InboundRequestID(r))
		return
	}

	if results == nil {
		results = []TMDBMovieDetail{}
	}

	api.WriteOK(w, http.StatusOK, results, httpx.InboundRequestID(r))
}

func parsePathID(path string, segmentIndex int) (int64, error) {
	parts := strings.Split(path, "/")
	if segmentIndex >= len(parts) {
		return 0, nil
	}
	idStr := parts[len(parts)-segmentIndex]
	return strconv.ParseInt(idStr, 10, 64)
}

func intParam(s string) int {
	if s == "" {
		return 0
	}
	v, _ := strconv.Atoi(s)
	return v
}

func decodeJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

var _ = decodeJSON