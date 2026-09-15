package queue

import (
	"context"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ExchangeAnalyticsEvents = "analytics.events"
	QueueAnalyticsClicks    = "analytics.clicks"
	RoutingKeyURLClicked    = "url.clicked"

	ExchangeAnalyticsDLX = "analytics.events.dlx"
	QueueAnalyticsDLQ    = "analytics.clicks.dlq"
)

type RabbitMQClient struct {
	conn      *amqp.Connection
	publishCh *amqp.Channel
	consumeCh *amqp.Channel
}

// NewRabbitMQClient connects to RabbitMQ, opens channels, and sets up the topology.
func NewRabbitMQClient(url string) (*RabbitMQClient, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	pubCh, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to open publish channel: %w", err)
	}

	subCh, err := conn.Channel()
	if err != nil {
		_ = pubCh.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("failed to open consume channel: %w", err)
	}

	client := &RabbitMQClient{
		conn:      conn,
		publishCh: pubCh,
		consumeCh: subCh,
	}

	if err := client.setupTopology(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to setup topology: %w", err)
	}

	return client, nil
}

func (c *RabbitMQClient) setupTopology() error {
	// 1. Setup Dead Letter Exchange & Queue
	err := c.publishCh.ExchangeDeclare(
		ExchangeAnalyticsDLX,
		"direct",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return err
	}

	_, err = c.publishCh.QueueDeclare(
		QueueAnalyticsDLQ,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return err
	}

	err = c.publishCh.QueueBind(
		QueueAnalyticsDLQ,
		RoutingKeyURLClicked,
		ExchangeAnalyticsDLX,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// 2. Setup Main Exchange & Queue with DLX args
	err = c.publishCh.ExchangeDeclare(
		ExchangeAnalyticsEvents,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	args := amqp.Table{
		"x-dead-letter-exchange":    ExchangeAnalyticsDLX,
		"x-dead-letter-routing-key": RoutingKeyURLClicked,
	}

	_, err = c.publishCh.QueueDeclare(
		QueueAnalyticsClicks,
		true,
		false,
		false,
		false,
		args,
	)
	if err != nil {
		return err
	}

	err = c.publishCh.QueueBind(
		QueueAnalyticsClicks,
		RoutingKeyURLClicked,
		ExchangeAnalyticsEvents,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	return nil
}

// Publish sends a message to the analytics exchange.
func (c *RabbitMQClient) Publish(ctx context.Context, payload []byte) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := c.publishCh.PublishWithContext(ctx,
		ExchangeAnalyticsEvents,
		RoutingKeyURLClicked,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         payload,
		})
	
	if err != nil {
		log.Printf("Failed to publish message: %v", err)
		return err
	}
	return nil
}

// Consume registers a consumer and returns a Go channel of deliveries.
func (c *RabbitMQClient) Consume(consumerName string) (<-chan amqp.Delivery, error) {
	return c.consumeCh.Consume(
		QueueAnalyticsClicks,
		consumerName,
		false, // auto-ack (we want manual ack)
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
}

func (c *RabbitMQClient) Close() {
	if c.publishCh != nil {
		_ = c.publishCh.Close()
	}
	if c.consumeCh != nil {
		_ = c.consumeCh.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
}
