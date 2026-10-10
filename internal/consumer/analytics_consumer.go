package consumer

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/DanielCahya/url-shortener/internal/metrics"
	"github.com/DanielCahya/url-shortener/internal/queue"
	"github.com/DanielCahya/url-shortener/internal/repository"
	"github.com/DanielCahya/url-shortener/internal/url"
	amqp "github.com/rabbitmq/amqp091-go"
)

type AnalyticsConsumer struct {
	rabbitMQ *queue.RabbitMQClient
	repo     repository.ClickEventRepository
}

func NewAnalyticsConsumer(rabbitMQ *queue.RabbitMQClient, repo repository.ClickEventRepository) *AnalyticsConsumer {
	return &AnalyticsConsumer{
		rabbitMQ: rabbitMQ,
		repo:     repo,
	}
}

func (c *AnalyticsConsumer) Start(ctx context.Context) {
	log.Println("Starting Analytics Consumer...")

	deliveries, err := c.rabbitMQ.Consume("analytics-worker-1")
	if err != nil {
		log.Fatalf("Failed to register consumer: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("Analytics Consumer shutting down...")
			return
		case d, ok := <-deliveries:
			if !ok {
				log.Println("RabbitMQ delivery channel closed")
				return
			}

			c.processDelivery(ctx, d)
		}
	}
}

func (c *AnalyticsConsumer) processDelivery(ctx context.Context, d amqp.Delivery) {
	var event url.ClickEvent
	if err := json.Unmarshal(d.Body, &event); err != nil {
		log.Printf("Failed to unmarshal click event payload: %v", err)
		metrics.AnalyticsEventFailedTotal.Inc()
		// Reject without requeue, it will go to DLQ
		_ = d.Reject(false)
		return
	}

	// Persist to database
	// Timeout to prevent hanging if DB is unresponsive
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := c.repo.Insert(dbCtx, event); err != nil {
		log.Printf("Failed to insert click event (EventID: %s): %v", event.EventID, err)
		metrics.AnalyticsEventFailedTotal.Inc()
		// Nack without requeue, we can rely on outbox publisher to retry if it wasn't marked published
		// Actually, if DB is down, it will go to DLQ.
		_ = d.Nack(false, false)
		return
	}

	// Fire Webhook if present
	if event.WebhookURL != nil && *event.WebhookURL != "" {
		go func(webhookUrl string, payload []byte) {
			req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, webhookUrl, bytes.NewBuffer(payload))
			if err == nil {
				req.Header.Set("Content-Type", "application/json")
				client := &http.Client{Timeout: 5 * time.Second}
				resp, err := client.Do(req)
				if err != nil {
					log.Printf("Failed to deliver webhook to %s: %v", webhookUrl, err)
				} else {
					_ = resp.Body.Close()
					log.Printf("Delivered webhook to %s (Status: %d)", webhookUrl, resp.StatusCode)
				}
			}
		}(*event.WebhookURL, d.Body)
	}

	// Success, acknowledge the message
	if err := d.Ack(false); err != nil {
		log.Printf("Failed to ACK message (EventID: %s): %v", event.EventID, err)
	} else {
		metrics.AnalyticsEventProcessedTotal.Inc()
	}
}
