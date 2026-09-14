package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/DanielCahya/url-shortener/internal/repository"
)

type CleanupWorker struct {
	repo     repository.IdempotencyRepository
	interval time.Duration
	logger   *slog.Logger
	stopCh   chan struct{}
}

func NewCleanupWorker(repo repository.IdempotencyRepository, interval time.Duration, logger *slog.Logger) *CleanupWorker {
	return &CleanupWorker{
		repo:     repo,
		interval: interval,
		logger:   logger,
		stopCh:   make(chan struct{}),
	}
}

func (w *CleanupWorker) Start() {
	w.logger.Info("starting idempotency cleanup worker", slog.String("interval", w.interval.String()))
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			w.logger.Info("stopping idempotency cleanup worker")
			return
		case <-ticker.C:
			w.cleanup()
		}
	}
}

func (w *CleanupWorker) Stop() {
	close(w.stopCh)
}

func (w *CleanupWorker) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	now := time.Now()
	deleted, err := w.repo.DeleteExpired(ctx, now)
	if err != nil {
		w.logger.Error("failed to delete expired idempotency keys", slog.String("error", err.Error()))
		return
	}

	if deleted > 0 {
		w.logger.Info("deleted expired idempotency keys", slog.Int64("count", deleted))
	}
}
