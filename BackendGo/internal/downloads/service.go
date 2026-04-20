package downloads

import (
	"context"
	"fmt"
	"strings"
)

type TaskStore interface {
	CreateTask(ctx context.Context, searchResultID int64) (*DownloadTask, error)
	GetTask(ctx context.Context, id int64) (*DownloadTask, error)
	SaveTask(ctx context.Context, task DownloadTask) (*DownloadTask, error)
	ListTasks(ctx context.Context, limit, offset int) ([]DownloadTask, int64, error)
	RecordTransition(ctx context.Context, taskID int64, from *DownloadTaskStatus, to DownloadTaskStatus, reason string) error
}

type TorrentDeleter interface {
	Delete(ctx context.Context, hash string) error
}

type EventPublisher interface {
	PublishProgress(ctx context.Context, task DownloadTask) error
	PublishFailed(ctx context.Context, taskID int64, reason string) error
	PublishCompleted(ctx context.Context, taskID int64, filePath string) error
}

type Service struct {
	repo    TaskStore
	torrent TorrentDeleter
	events  EventPublisher
}

func NewService(repo TaskStore, torrent TorrentDeleter, events EventPublisher) *Service {
	return &Service{repo: repo, torrent: torrent, events: events}
}

func (s *Service) CreateTask(ctx context.Context, searchResultID int64) (*DownloadTask, error) {
	if searchResultID <= 0 {
		return nil, ValidationError{Field: "searchResultId", Message: "Search result ID must be positive"}
	}
	task, err := s.repo.CreateTask(ctx, searchResultID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.RecordTransition(ctx, task.ID, nil, StatusRequested, ""); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *Service) GetTask(ctx context.Context, id int64) (*DownloadTask, error) {
	return s.repo.GetTask(ctx, id)
}

func (s *Service) ListTasks(ctx context.Context, limit, offset int) ([]DownloadTask, int64, error) {
	return s.repo.ListTasks(ctx, limit, offset)
}

func (s *Service) StartSearching(ctx context.Context, id int64) (*DownloadTask, error) {
	return s.transition(ctx, id, StatusSearching, "", func(task *DownloadTask) {})
}

func (s *Service) MarkSearchReady(ctx context.Context, id int64) (*DownloadTask, error) {
	return s.transition(ctx, id, StatusSearchReady, "", func(task *DownloadTask) {})
}

func (s *Service) QueueTask(ctx context.Context, id int64, torrentHash string) (*DownloadTask, error) {
	return s.transition(ctx, id, StatusQueued, "", func(task *DownloadTask) {
		task.TorrentHash = strings.TrimSpace(torrentHash)
	})
}

func (s *Service) StartDownload(ctx context.Context, id int64) (*DownloadTask, error) {
	return s.transition(ctx, id, StatusDownloading, "", func(task *DownloadTask) {})
}

func (s *Service) UpdateProgress(ctx context.Context, id int64, progress float64, speed int64, peerCount int) (*DownloadTask, error) {
	task, err := s.repo.GetTask(ctx, id)
	if err != nil {
		return nil, err
	}
	if task.Status.IsTerminal() {
		return nil, fmt.Errorf("%w from %s", ErrInvalidTransition, task.Status)
	}
	task.Progress = clamp(progress, 0, 100)
	task.Speed = speed
	task.PeerCount = peerCount
	updated, err := s.repo.SaveTask(ctx, *task)
	if err != nil {
		return nil, err
	}
	if s.events != nil {
		_ = s.events.PublishProgress(ctx, *updated)
	}
	return updated, nil
}

func (s *Service) StartPostProcessing(ctx context.Context, id int64) (*DownloadTask, error) {
	return s.transition(ctx, id, StatusPostProcessing, "", func(task *DownloadTask) {})
}

func (s *Service) MarkStreamReady(ctx context.Context, id int64) (*DownloadTask, error) {
	return s.transition(ctx, id, StatusStreamReady, "", func(task *DownloadTask) {})
}

func (s *Service) MarkCompleted(ctx context.Context, id int64, filePath string) (*DownloadTask, error) {
	task, err := s.transition(ctx, id, StatusCompleted, "", func(task *DownloadTask) {
		task.Progress = 100
	})
	if err != nil {
		return nil, err
	}
	if s.events != nil {
		_ = s.events.PublishCompleted(ctx, id, filePath)
	}
	return task, nil
}

func (s *Service) MarkFailed(ctx context.Context, id int64, reason string) (*DownloadTask, error) {
	task, err := s.transition(ctx, id, StatusFailed, reason, func(task *DownloadTask) {})
	if err != nil {
		return nil, err
	}
	if s.events != nil {
		_ = s.events.PublishFailed(ctx, id, reason)
	}
	return task, nil
}

func (s *Service) CancelTask(ctx context.Context, id int64) (*DownloadTask, error) {
	task, err := s.repo.GetTask(ctx, id)
	if err != nil {
		return nil, err
	}
	if task.Status.IsTerminal() {
		return nil, fmt.Errorf("%w from %s", ErrInvalidTransition, task.Status)
	}
	if s.torrent != nil && strings.TrimSpace(task.TorrentHash) != "" {
		_ = s.torrent.Delete(ctx, task.TorrentHash)
	}
	from := task.Status
	task.Status = StatusCancelled
	updated, err := s.repo.SaveTask(ctx, *task)
	if err != nil {
		return nil, err
	}
	if err := s.repo.RecordTransition(ctx, id, &from, StatusCancelled, ""); err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Service) transition(ctx context.Context, id int64, to DownloadTaskStatus, reason string, mutate func(*DownloadTask)) (*DownloadTask, error) {
	task, err := s.repo.GetTask(ctx, id)
	if err != nil {
		return nil, err
	}
	from := task.Status
	if !from.CanTransitionTo(to) {
		return nil, fmt.Errorf("%w from %s to %s", ErrInvalidTransition, from, to)
	}
	if mutate != nil {
		mutate(task)
	}
	task.Status = to
	updated, err := s.repo.SaveTask(ctx, *task)
	if err != nil {
		return nil, err
	}
	if err := s.repo.RecordTransition(ctx, id, &from, to, reason); err != nil {
		return nil, err
	}
	return updated, nil
}

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
