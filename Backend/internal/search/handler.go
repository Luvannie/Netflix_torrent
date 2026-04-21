package search

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
		"POST /api/v1/search/jobs":                         http.HandlerFunc(h.createJob),
		"GET /api/v1/search/jobs":                          http.HandlerFunc(h.listJobs),
		"GET /api/v1/search/jobs/{id}":                     http.HandlerFunc(h.getJob),
		"POST /api/v1/search/jobs/{id}/process":            http.HandlerFunc(h.processJob),
		"DELETE /api/v1/search/jobs/{id}":                  http.HandlerFunc(h.cancelJob),
	}
}

func (h *Handler) createJob(w http.ResponseWriter, r *http.Request) {
	var req CreateSearchJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil, httpx.InboundRequestID(r))
		return
	}

	id, err := h.service.CreateJob(r.Context(), req.Query)
	if err != nil {
		if ve, ok := err.(*ValidationError); ok {
			api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", ve.Message, nil, httpx.InboundRequestID(r))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Failed to create search job", nil, httpx.InboundRequestID(r))
		return
	}

	api.WriteOK(w, http.StatusOK, id, httpx.InboundRequestID(r))
}

func (h *Handler) listJobs(w http.ResponseWriter, r *http.Request) {
	page, size := pagination.Parse(r)
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	limit, offset := pagination.LimitOffset(page, size)

	jobs, total, err := h.service.ListJobs(r.Context(), query, limit, offset)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Failed to list search jobs", nil, httpx.InboundRequestID(r))
		return
	}

	if jobs == nil {
		jobs = []SearchJob{}
	}

	api.WriteOK(w, http.StatusOK, pagination.New(jobs, page, size, total), httpx.InboundRequestID(r))
}

func (h *Handler) getJob(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathID(r.URL.Path)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid job ID", nil, httpx.InboundRequestID(r))
		return
	}

	job, err := h.service.GetJob(r.Context(), id)
	if err != nil {
		api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Search job not found", nil, httpx.InboundRequestID(r))
		return
	}

	api.WriteOK(w, http.StatusOK, job, httpx.InboundRequestID(r))
}

func (h *Handler) processJob(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathID(r.URL.Path)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid job ID", nil, httpx.InboundRequestID(r))
		return
	}

	if err := h.service.ProcessJob(r.Context(), id); err != nil {
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Failed to process search job", nil, httpx.InboundRequestID(r))
		return
	}

	api.WriteOK(w, http.StatusOK, nil, httpx.InboundRequestID(r))
}

func (h *Handler) cancelJob(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathID(r.URL.Path)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid job ID", nil, httpx.InboundRequestID(r))
		return
	}

	if err := h.service.CancelJob(r.Context(), id); err != nil {
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Failed to cancel search job", nil, httpx.InboundRequestID(r))
		return
	}

	api.WriteOK(w, http.StatusOK, nil, httpx.InboundRequestID(r))
}

func parsePathID(path string) (int64, error) {
	parts := strings.Split(path, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if v, err := strconv.ParseInt(parts[i], 10, 64); err == nil {
			return v, nil
		}
	}
	return 0, nil
}
