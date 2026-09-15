package worker

import (
	"context"
	"log"
	"time"

	"github.com/DanielCahya/url-shortener/internal/queue"
	"github.com/DanielCahya/url-shortener/internal/repository"
	"github.com/jackc/pgx/v5"
)

// OutboxPublisher is a background worker that polls the outbox_events table
// and publishes events to RabbitMQ.
type OutboxPublisher struct {
	repo         repository.OutboxRepository
	rabbitMQ     *queue.RabbitMQClient
	batchSize    int
	pollInterval time.Duration
}

func NewOutboxPublisher(repo repository.OutboxRepository, rabbitMQ *queue.RabbitMQClient) *OutboxPublisher {
	return &OutboxPublisher{
		repo:         repo,
		rabbitMQ:     rabbitMQ,
		batchSize:    100,
		pollInterval: 2 * time.Second,
	}
}

// Start begins polling for outbox events on a ticker.
func (p *OutboxPublisher) Start(ctx context.Context) {
	log.Println("Starting Outbox Publisher worker...")
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Outbox Publisher worker shutting down...")
			return
		case <-ticker.C:
			p.processBatch(ctx)
		}
	}
}

func (p *OutboxPublisher) processBatch(ctx context.Context) {
	err := p.repo.WithTransaction(ctx, func(tx pgx.Tx) error {
		events, err := p.repo.FetchUnpublishedTx(ctx, tx, p.batchSize)
		if err != nil {
			return err
		}

		if len(events) == 0 {
			return nil
		}

		var publishedIDs []string
		var failedIDs []string

		for _, event := range events {
			// Publish to RabbitMQ
			err := p.rabbitMQ.Publish(ctx, event.Payload)
			if err != nil {
				log.Printf("Failed to publish event %s: %v", event.ID, err)
				failedIDs = append(failedIDs, event.ID)
			} else {
				publishedIDs = append(publishedIDs, event.ID)
			}
		}

		if len(publishedIDs) > 0 {
			if err := p.repo.MarkPublishedTx(ctx, tx, publishedIDs); err != nil {
				log.Printf("Failed to mark events as published: %v", err)
			}
		}

		if len(failedIDs) > 0 {
			if err := p.repo.IncrementAttemptsTx(ctx, tx, failedIDs); err != nil {
				log.Printf("Failed to increment attempts for events: %v", err)
			}
		}

		return nil
	})

	if err != nil {
		log.Printf("Error processing outbox batch: %v", err)
	}
}
