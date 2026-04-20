package search

import (
	"context"
	"log/slog"
	"time"
)

type Worker struct {
	repo     *Repository
	service  *Service
	interval time.Duration
	logger   *slog.Logger
	stop     chan struct{}
}

func NewWorker(repo *Repository, service *Service, interval time.Duration, logger *slog.Logger) *Worker {
	return &Worker{
		repo:     repo,
		service:  service,
		interval: interval,
		logger:   logger,
		stop:     make(chan struct{}),
	}
}

func (w *Worker) Start(ctx context.Context) {
	w.logger.Info("search worker starting")
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.processJobs(ctx)
		case <-w.stop:
			w.logger.Info("search worker stopped")
			return
		case <-ctx.Done():
			return
		}
	}
}

func (w *Worker) Stop() {
	close(w.stop)
}

func (w *Worker) processJobs(ctx context.Context) {
	jobs, err := w.repo.GetRequestedJobs(ctx, 10)
	if err != nil {
		w.logger.Error("failed to fetch requested jobs", "error", err)
		return
	}

	for _, job := range jobs {
		w.logger.Info("processing search job", "id", job.ID, "query", job.Query)

		if err := w.service.ProcessJob(ctx, job.ID); err != nil {
			w.logger.Error("failed to process search job", "id", job.ID, "error", err)
		} else {
			w.logger.Info("search job completed", "id", job.ID)
		}
	}
}