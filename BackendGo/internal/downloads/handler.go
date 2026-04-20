package downloads

import (
	"encoding/json"
	"errors"
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
		"POST /api/v1/downloads":             http.HandlerFunc(h.create),
		"GET /api/v1/downloads":              http.HandlerFunc(h.list),
		"GET /api/v1/downloads/{id}":         http.HandlerFunc(h.get),
		"POST /api/v1/downloads/{id}/cancel": http.HandlerFunc(h.cancel),
	}
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req CreateDownloadTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil, httpx.InboundRequestID(r))
		return
	}
	if req.SearchResultID == nil || *req.SearchResultID <= 0 {
		api.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Search result ID is required", []api.ErrorDetail{{Field: "searchResultId", Message: "Search result ID is required"}}, httpx.InboundRequestID(r))
		return
	}

	task, err := h.service.CreateTask(r.Context(), *req.SearchResultID)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Failed to create download task", nil, httpx.InboundRequestID(r))
		return
	}
	api.WriteOK(w, http.StatusOK, task, httpx.InboundRequestID(r))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	page, size := pagination.Parse(r)
	limit, offset := pagination.LimitOffset(page, size)
	tasks, total, err := h.service.ListTasks(r.Context(), limit, offset)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Failed to list download tasks", nil, httpx.InboundRequestID(r))
		return
	}
	if tasks == nil {
		tasks = []DownloadTask{}
	}
	api.WriteOK(w, http.StatusOK, pagination.New(tasks, page, size, total), httpx.InboundRequestID(r))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathID(r.URL.Path)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid download task ID", nil, httpx.InboundRequestID(r))
		return
	}
	task, err := h.service.GetTask(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Download task not found", nil, httpx.InboundRequestID(r))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Failed to get download task", nil, httpx.InboundRequestID(r))
		return
	}
	api.WriteOK(w, http.StatusOK, task, httpx.InboundRequestID(r))
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathID(r.URL.Path)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid download task ID", nil, httpx.InboundRequestID(r))
		return
	}
	task, err := h.service.CancelTask(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Download task not found", nil, httpx.InboundRequestID(r))
			return
		}
		if errors.Is(err, ErrInvalidTransition) {
			api.WriteError(w, http.StatusConflict, "INVALID_STATE", err.Error(), nil, httpx.InboundRequestID(r))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Failed to cancel download task", nil, httpx.InboundRequestID(r))
		return
	}
	api.WriteOK(w, http.StatusOK, task, httpx.InboundRequestID(r))
}

func parsePathID(path string) (int64, error) {
	parts := strings.Split(path, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		part := strings.TrimSpace(parts[i])
		if part == "" || part == "cancel" {
			continue
		}
		if id, err := strconv.ParseInt(part, 10, 64); err == nil {
			return id, nil
		}
	}
	return 0, strconv.ErrSyntax
}
