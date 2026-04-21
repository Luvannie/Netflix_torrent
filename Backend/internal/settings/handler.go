package settings

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/netflixtorrent/backend-go/internal/api"
	"github.com/netflixtorrent/backend-go/internal/httpx"
)

type Handler struct {
	repo        *Repository
	pathResolver *PathResolver
}

func NewHandler(repo *Repository, pathResolver *PathResolver) *Handler {
	return &Handler{repo: repo, pathResolver: pathResolver}
}

func (h *Handler) Routes() map[string]http.Handler {
	return map[string]http.Handler{
		"GET /api/v1/settings/storage-profiles":           http.HandlerFunc(h.list),
		"GET /api/v1/settings/storage-profiles/{id}":      http.HandlerFunc(h.getByID),
		"POST /api/v1/settings/storage-profiles":           http.HandlerFunc(h.create),
		"PUT /api/v1/settings/storage-profiles/{id}":      http.HandlerFunc(h.update),
		"DELETE /api/v1/settings/storage-profiles/{id}":    http.HandlerFunc(h.delete),
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.repo.List(r.Context())
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Failed to list storage profiles", nil, httpx.InboundRequestID(r))
		return
	}
	if profiles == nil {
		profiles = []StorageProfile{}
	}
	api.WriteOK(w, http.StatusOK, profiles, httpx.InboundRequestID(r))
}

func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r.URL.Path)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid storage profile ID", nil, httpx.InboundRequestID(r))
		return
	}

	profile, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if isNotFound(err) {
			api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Storage profile not found with ID: "+strconv.FormatInt(id, 10), nil, httpx.InboundRequestID(r))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Failed to get storage profile", nil, httpx.InboundRequestID(r))
		return
	}
	api.WriteOK(w, http.StatusOK, profile, httpx.InboundRequestID(r))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req CreateStorageProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil, httpx.InboundRequestID(r))
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		api.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Name is required", []api.ErrorDetail{{Field: "name", Message: "Name is required"}}, httpx.InboundRequestID(r))
		return
	}
	if len(req.Name) > 100 {
		api.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Name must not exceed 100 characters", []api.ErrorDetail{{Field: "name", Message: "Name must not exceed 100 characters"}}, httpx.InboundRequestID(r))
		return
	}
	if strings.TrimSpace(req.BasePath) == "" {
		api.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Base path is required", []api.ErrorDetail{{Field: "basePath", Message: "Base path is required"}}, httpx.InboundRequestID(r))
		return
	}

	if req.Priority != nil && *req.Priority < 0 {
		api.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Priority must be zero or positive", []api.ErrorDetail{{Field: "priority", Message: "Priority must be zero or positive"}}, httpx.InboundRequestID(r))
		return
	}

	if h.pathResolver != nil {
		if _, err := h.pathResolver.ResolveAndValidate(req.BasePath); err != nil {
			api.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error(), nil, httpx.InboundRequestID(r))
			return
		}
	}

	profile, err := h.repo.Create(r.Context(), req)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Failed to create storage profile", nil, httpx.InboundRequestID(r))
		return
	}
	api.WriteOK(w, http.StatusOK, profile, httpx.InboundRequestID(r))
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r.URL.Path)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid storage profile ID", nil, httpx.InboundRequestID(r))
		return
	}

	var req UpdateStorageProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil, httpx.InboundRequestID(r))
		return
	}

	profile, err := h.repo.Update(r.Context(), id, req)
	if err != nil {
		if isNotFound(err) {
			api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Storage profile not found with ID: "+strconv.FormatInt(id, 10), nil, httpx.InboundRequestID(r))
			return
		}
		var validationErr ValidationError
		if _, ok := err.(ValidationError); ok {
			api.WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error(), nil, httpx.InboundRequestID(r))
			return
		}
		_ = validationErr
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Failed to update storage profile", nil, httpx.InboundRequestID(r))
		return
	}
	api.WriteOK(w, http.StatusOK, profile, httpx.InboundRequestID(r))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r.URL.Path)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid storage profile ID", nil, httpx.InboundRequestID(r))
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		if isNotFound(err) {
			api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Storage profile not found with ID: "+strconv.FormatInt(id, 10), nil, httpx.InboundRequestID(r))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Failed to delete storage profile", nil, httpx.InboundRequestID(r))
		return
	}
	api.WriteOK(w, http.StatusOK, nil, httpx.InboundRequestID(r))
}

func (h *Handler) parseID(path string) (int64, error) {
	parts := strings.Split(path, "/")
	var parseErr error
	for i := len(parts) - 1; i >= 0; i-- {
		if id, err := strconv.ParseInt(parts[i], 10, 64); err == nil {
			return id, nil
		} else {
			parseErr = err
		}
	}
	return 0, parseErr
}

func isNotFound(err error) bool {
	_, ok := err.(NotFoundError)
	return ok
}