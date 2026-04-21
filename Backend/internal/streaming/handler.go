package streaming

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/netflixtorrent/backend-go/internal/api"
	"github.com/netflixtorrent/backend-go/internal/app"
	"github.com/netflixtorrent/backend-go/internal/httpx"
	"github.com/netflixtorrent/backend-go/internal/library"
)

type Repository interface {
	GetMediaFileByID(ctx context.Context, id int64) (*library.MediaFile, error)
}

type FFProbeRunner interface {
	Run(ctx context.Context, path string) (*MediaInfo, error)
}

type Service struct {
	repo      Repository
	ffprobe   FFProbeRunner
	storagePath string
}

func NewService(repo Repository, ffprobePath string, storagePath string) *Service {
	return &Service{
		repo:        repo,
		ffprobe:     &ffprobeCLI{path: ffprobePath},
		storagePath: storagePath,
	}
}

type MediaInfo struct {
	Duration  float64 `json:"duration"`
	Width     int     `json:"width"`
	Height    int     `json:"height"`
	Codec     string  `json:"codec"`
	Container string  `json:"container"`
}

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Routes() map[string]http.Handler {
	return map[string]http.Handler{
		"GET /api/v1/streams/{id}": http.HandlerFunc(h.stream),
		"GET /api/v1/streams/{id}/info": http.HandlerFunc(h.info),
	}
}

func (h *Handler) stream(w http.ResponseWriter, r *http.Request) {
	idStr := app.PathValue(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid media file ID", nil, httpx.InboundRequestID(r))
		return
	}

	file, err := h.service.repo.GetMediaFileByID(r.Context(), id)
	if err != nil {
		api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Media file not found", nil, httpx.InboundRequestID(r))
		return
	}

	if !filepath.IsAbs(file.FilePath) {
		file.FilePath = filepath.Join(h.service.storagePath, file.FilePath)
	}

	if _, err := os.Stat(file.FilePath); os.IsNotExist(err) {
		api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "File not found on disk", nil, httpx.InboundRequestID(r))
		return
	}

	http.ServeFile(w, r, file.FilePath)
}

func (h *Handler) info(w http.ResponseWriter, r *http.Request) {
	idStr := app.PathValue(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid media file ID", nil, httpx.InboundRequestID(r))
		return
	}

	file, err := h.service.repo.GetMediaFileByID(r.Context(), id)
	if err != nil {
		api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Media file not found", nil, httpx.InboundRequestID(r))
		return
	}

	api.WriteOK(w, http.StatusOK, file, httpx.InboundRequestID(r))
}

type ffprobeCLI struct {
	path string
}

func (f *ffprobeCLI) Run(ctx context.Context, path string) (*MediaInfo, error) {
	cmd := exec.CommandContext(ctx, f.path, "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", path)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parseFFProbeOutput(output)
}

func parseFFProbeOutput(data []byte) (*MediaInfo, error) {
	return &MediaInfo{
		Duration:  0,
		Width:     0,
		Height:    0,
		Codec:     "unknown",
		Container: "unknown",
	}, nil
}

type StreamHandler struct {
	repo    Repository
	baseURL string
}

func NewStreamHandler(repo Repository, baseURL string) *StreamHandler {
	return &StreamHandler{repo: repo, baseURL: baseURL}
}

func (h *StreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	idStr := app.PathValue(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid media file ID", nil, httpx.InboundRequestID(r))
		return
	}

	file, err := h.repo.GetMediaFileByID(r.Context(), id)
	if err != nil {
		api.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Media file not found", nil, httpx.InboundRequestID(r))
		return
	}

	if h.baseURL != "" {
		streamURL := fmt.Sprintf("%s/api/v1/streams/%d/file", h.baseURL, file.ID)
		api.WriteOK(w, http.StatusOK, map[string]string{"url": streamURL}, "")
		return
	}

	reader, err := os.Open(file.FilePath)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Failed to open file", nil, httpx.InboundRequestID(r))
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "video/mp4")
	http.ServeContent(w, r, filepath.Base(file.FilePath), time.Now(), reader)
}

type rangeReader struct {
	f   *os.File
	end int64
}

func (rr *rangeReader) Read(p []byte) (int, error) {
	if rr.end > 0 {
		n, err := rr.f.Read(p)
		rr.end -= int64(n)
		if rr.end <= 0 {
			return n, io.EOF
		}
		return n, err
	}
	return rr.f.Read(p)
}
