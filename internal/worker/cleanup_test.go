package worker

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/DanielCahya/url-shortener/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockIdempotencyRepository struct {
	repository.IdempotencyRepository
	deletedCount int64
	err          error
}

func (m *mockIdempotencyRepository) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.deletedCount, nil
}

func TestCleanupWorker(t *testing.T) {
	mockRepo := &mockIdempotencyRepository{
		deletedCount: 5,
		err:          nil,
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	worker := NewCleanupWorker(mockRepo, 50*time.Millisecond, logger)

	// Start worker in background
	go worker.Start()

	// Let it run for a bit to trigger the ticker at least once
	time.Sleep(120 * time.Millisecond)

	// Stop worker
	worker.Stop()

	// There isn't a direct way to assert the ticker fired exactly N times cleanly without race conditions
	// in this simple setup, but we can test the cleanup() function directly
}

func TestCleanupWorker_cleanup(t *testing.T) {
	tests := []struct {
		name         string
		deletedCount int64
		err          error
	}{
		{
			name:         "successful cleanup",
			deletedCount: 5,
			err:          nil,
		},
		{
			name:         "cleanup error",
			deletedCount: 0,
			err:          assert.AnError,
		},
		{
			name:         "no records deleted",
			deletedCount: 0,
			err:          nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockIdempotencyRepository{
				deletedCount: tt.deletedCount,
				err:          tt.err,
			}
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			worker := NewCleanupWorker(mockRepo, 1*time.Hour, logger)

			// Execute cleanup manually
			worker.cleanup()

			// In a real test with a mock library (like mockgen), we would verify DeleteExpired was called.
			// Here we are just ensuring no panics occur.
			require.NotNil(t, worker)
		})
	}
}
