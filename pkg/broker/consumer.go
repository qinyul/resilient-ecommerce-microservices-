package broker

import (
	"context"
	"log/slog"
	"time"

	"github.com/qinyul/resilient-ecommerce-microservices/pkg/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Handler defines the function signature for processing an AMQP delivery.
// It returns an error and a boolean indicating if the message should be requeued.
type Handler func(ctx context.Context, d amqp.Delivery) (err error, requeue bool)

// EventConsumer represents a generic RabbitMQ consumer.
type EventConsumer struct {
	rmqClient    *rabbitmq.Client
	queueName    string
	exchangeName string
	routingKey   string
	consumerName string
	handler      Handler
}

// NewEventConsumer creates a new instance of EventConsumer.
func NewEventConsumer(
	rmqClient *rabbitmq.Client,
	queueName string,
	exchangeName string,
	routingKey string,
	consumerName string,
	handler Handler,
) *EventConsumer {
	return &EventConsumer{
		rmqClient:    rmqClient,
		queueName:    queueName,
		exchangeName: exchangeName,
		routingKey:   routingKey,
		consumerName: consumerName,
		handler:      handler,
	}
}

// Start begins consuming messages from the queue.
func (c *EventConsumer) Start(ctx context.Context) {
	slog.Info("Starting Event Consumer...", 
		"queue", c.queueName, 
		"exchange", c.exchangeName, 
		"routing_key", c.routingKey)

	for {
		if c.rmqClient.IsConnected() {
			// Ensure Exchange exists
			err := c.rmqClient.ExchangeDeclare(c.exchangeName, "topic", true, false)
			if err != nil {
				slog.Warn("Failed to declare exchange, retrying...", "error", err)
				goto retry
			}

			// Declare Queue
			// Note: In a production system, you'd add x-dead-letter-exchange here
			_, err = c.rmqClient.QueueDeclare(c.queueName, true, false)
			if err != nil {
				slog.Warn("Failed to declare queue, retrying...", "error", err)
				goto retry
			}

			// Bind Queue
			err = c.rmqClient.QueueBind(c.queueName, c.routingKey, c.exchangeName)
			if err != nil {
				slog.Warn("Failed to bind queue, retrying...", "error", err)
				goto retry
			}

			// Start Consuming
			msgs, err := c.rmqClient.Consume(c.queueName, c.consumerName, false)
			if err != nil {
				slog.Warn("Failed to start consuming, retrying...", "error", err)
				goto retry
			}

			slog.Info("Consumer is ready", "queue", c.queueName)
			
			// Process messages
			for {
				select {
				case <-ctx.Done():
					return
				case d, ok := <-msgs:
					if !ok {
						slog.Warn("Message channel closed, reconnecting...")
						goto retry
					}
					
					// Process message
					err, requeue := c.handler(ctx, d)
					if err != nil {
						slog.Error("Handler error", "error", err, "queue", c.queueName, "requeue", requeue)
						// If requeue is false and a DLX is configured, it goes to Dead Letter
						if nackErr := d.Nack(false, requeue); nackErr != nil {
							slog.Error("Failed to nack message", "error", nackErr)
						}
					} else {
						if ackErr := d.Ack(false); ackErr != nil {
							slog.Error("Failed to ack message", "error", ackErr, "queue", c.queueName)
						}
					}
				}
			}
		}

	retry:
		select {
		case <-ctx.Done():
			slog.Info("Consumer shutting down...", "queue", c.queueName)
			return
		case <-time.After(5 * time.Second):
			// Loop and check connection again
		}
	}
}
