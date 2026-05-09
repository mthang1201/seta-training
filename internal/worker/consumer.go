package worker

import (
	"context"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/seta-training/core/internal/infrastructure"
)

type EventConsumer struct {
	conn *amqp.Connection
}

func NewEventConsumer(conn *amqp.Connection) *EventConsumer {
	return &EventConsumer{conn: conn}
}

func (c *EventConsumer) Start(ctx context.Context) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return err
	}

	// Declare Team Queue
	teamQueue, err := ch.QueueDeclare(
		"team_audit_queue",
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return err
	}

	err = ch.QueueBind(
		teamQueue.Name,
		"team.#", // bind all team events
		infrastructure.TeamExchange,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// Declare Asset Queue
	assetQueue, err := ch.QueueDeclare(
		"asset_audit_queue",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	err = ch.QueueBind(
		assetQueue.Name,
		"asset.#", // bind all asset events
		infrastructure.AssetExchange,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// Consume Team events
	teamMsgs, err := ch.Consume(
		teamQueue.Name,
		"team_consumer",
		false, // auto-ack (manual ack for reliability)
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return err
	}

	// Consume Asset events
	assetMsgs, err := ch.Consume(
		assetQueue.Name,
		"asset_consumer",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case msg, ok := <-teamMsgs:
				if !ok {
					return
				}
				slog.Info("Received Team Event", "routing_key", msg.RoutingKey, "body", string(msg.Body))
				_ = msg.Ack(false)
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case msg, ok := <-assetMsgs:
				if !ok {
					return
				}
				slog.Info("Received Asset Event", "routing_key", msg.RoutingKey, "body", string(msg.Body))
				_ = msg.Ack(false)
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}
