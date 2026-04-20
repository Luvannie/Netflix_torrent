package downloads

import (
	"errors"
	"time"
)

type DownloadTaskStatus string

const (
	StatusRequested      DownloadTaskStatus = "REQUESTED"
	StatusSearching      DownloadTaskStatus = "SEARCHING"
	StatusSearchReady    DownloadTaskStatus = "SEARCH_READY"
	StatusQueued         DownloadTaskStatus = "QUEUED"
	StatusDownloading    DownloadTaskStatus = "DOWNLOADING"
	StatusPostProcessing DownloadTaskStatus = "POST_PROCESSING"
	StatusStreamReady    DownloadTaskStatus = "STREAM_READY"
	StatusCompleted      DownloadTaskStatus = "COMPLETED"
	StatusFailed         DownloadTaskStatus = "FAILED"
	StatusCancelled      DownloadTaskStatus = "CANCELLED"
)

var (
	ErrTaskNotFound      = errors.New("download task not found")
	ErrInvalidTransition = errors.New("invalid download task state transition")
)

type DownloadTask struct {
	ID             int64              `json:"id"`
	SearchResultID int64              `json:"searchResultId"`
	TorrentHash    string             `json:"torrentHash"`
	Status         DownloadTaskStatus `json:"status"`
	Progress       float64            `json:"progress"`
	Speed          int64              `json:"speed"`
	PeerCount      int                `json:"peerCount"`
	CreatedAt      time.Time          `json:"createdAt"`
	UpdatedAt      time.Time          `json:"updatedAt"`
}

type DownloadStateTransition struct {
	ID             int64
	DownloadTaskID int64
	FromStatus     *DownloadTaskStatus
	ToStatus       DownloadTaskStatus
	Timestamp      time.Time
	Reason         string
}

type CreateDownloadTaskRequest struct {
	SearchResultID *int64 `json:"searchResultId"`
}

type TorrentStatus struct {
	Hash          string
	Name          string
	Progress      float64
	DownloadSpeed int64
	UploadSpeed   int64
	PeerCount     int
	State         string
	SavePath      string
	ContentPath   string
	TotalSize     int64
}

func (s DownloadTaskStatus) String() string {
	return string(s)
}

func (s DownloadTaskStatus) IsTerminal() bool {
	return s == StatusCompleted || s == StatusFailed || s == StatusCancelled
}

func (s DownloadTaskStatus) CanTransitionTo(target DownloadTaskStatus) bool {
	if s.IsTerminal() {
		return false
	}
	if target.IsTerminal() {
		return true
	}

	switch s {
	case StatusRequested:
		return target == StatusSearching
	case StatusSearching:
		return target == StatusSearchReady
	case StatusSearchReady:
		return target == StatusQueued
	case StatusQueued:
		return target == StatusDownloading
	case StatusDownloading:
		return target == StatusPostProcessing
	case StatusPostProcessing:
		return target == StatusStreamReady
	case StatusStreamReady:
		return target == StatusCompleted
	default:
		return false
	}
}

func (s TorrentStatus) IsComplete() bool {
	return s.Progress >= 100 || s.State == "seeding"
}
