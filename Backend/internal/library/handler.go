package library

import (
	"net/http"
	"strconv"

	"github.com/netflixtorrent/backend-go/internal/api"
	"github.com/netflixtorrent/backend-go/internal/app"
	"github.com/netflixtorrent/backend-go/internal/httpx"
	"github.com/netflixtorrent/backend-go/internal/pagination"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) Routes() map[string]http.Handler {
	return map[string]http.Handler{
		"GET /api/v1/library":                  http.HandlerFunc(h.list),
		"GET /api/v1/library/{id}":            http.HandlerFunc(h.get),
		"DELETE /api/v1/library/{id}":        http.HandlerFunc(h.delete),
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	page, size := pagination.Parse(r)
	mediaType := r.URL.Query().Get("type")
	limit, offset := pagination.LimitOffset(page, size)

	items, total, err := h.repo.ListMediaItems(r.Context(), mediaType, limit, offset)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Failed to list library", nil, httpx.InboundRequestID(r))
		return
	}

	if items == nil {
		items = []MediaItem{}
	}

	api.WriteOK(w, http.StatusOK, pagination.New(items, page, size, total), httpx.InboundRequestID(r))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	idStr := app.PathValue(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid media item ID", nil, httpx.InboundRequestID(r))
		return
	}

	item, err := h.repo.GetMediaItemByID(r.Context(), id)
	if err != nil {
		api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Media item not found", nil, httpx.InboundRequestID(r))
		return
	}

	api.WriteOK(w, http.StatusOK, item, httpx.InboundRequestID(r))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	idStr := app.PathValue(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid media item ID", nil, httpx.InboundRequestID(r))
		return
	}

	if err := h.repo.DeleteMediaItem(r.Context(), id); err != nil {
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Failed to delete media item", nil, httpx.InboundRequestID(r))
		return
	}

	api.WriteOK(w, http.StatusOK, nil, httpx.InboundRequestID(r))
}
