package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/seta-training/internal/config"
	"github.com/seta-training/internal/domain"
)

const (
	TeamExchange  = "team.topic"
	AssetExchange = "asset.topic"
)

type rabbitMQPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewRabbitMQPublisher(cfg *config.Config) (domain.EventPublisher, *amqp.Connection, error) {
	conn, err := amqp.Dial(cfg.RabbitMQURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	// Declare Exchanges
	err = ch.ExchangeDeclare(
		TeamExchange, // name
		"topic",      // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to declare team exchange: %w", err)
	}

	err = ch.ExchangeDeclare(
		AssetExchange, // name
		"topic",       // type
		true,          // durable
		false,         // auto-deleted
		false,         // internal
		false,         // no-wait
		nil,           // arguments
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to declare asset exchange: %w", err)
	}

	return &rabbitMQPublisher{
		conn:    conn,
		channel: ch,
	}, conn, nil
}

func (p *rabbitMQPublisher) PublishTeamEvent(ctx context.Context, eventType string, payload interface{}) error {
	routingKey := fmt.Sprintf("team.%s", eventType)
	return p.publish(ctx, TeamExchange, routingKey, eventType, payload)
}

func (p *rabbitMQPublisher) PublishAssetEvent(ctx context.Context, eventType string, payload interface{}) error {
	routingKey := fmt.Sprintf("asset.%s", eventType)
	return p.publish(ctx, AssetExchange, routingKey, eventType, payload)
}

func (p *rabbitMQPublisher) publish(ctx context.Context, exchange, routingKey, eventType string, payload interface{}) error {
	event := domain.Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Timestamp: time.Now(),
		Payload:   payload,
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	err = p.channel.PublishWithContext(ctx,
		exchange,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent, // Persistent messages
			ContentType:  "application/json",
			Body:         body,
			MessageId:    event.ID,
			Timestamp:    event.Timestamp,
			Type:         eventType,
		})

	if err != nil {
		slog.Error("Failed to publish message", "exchange", exchange, "routingKey", routingKey, "error", err)
		return err
	}

	slog.Info("Event published", "exchange", exchange, "routingKey", routingKey, "eventID", event.ID)
	return nil
}
