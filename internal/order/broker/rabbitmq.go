package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/qinyul/resilient-ecommerce-microservices/internal/order/domain"
	"github.com/qinyul/resilient-ecommerce-microservices/pkg/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

type rabbitMQBroker struct {
	client *rabbitmq.Client
}

const (
	OrderExchange   = "order_exchange"
	OrderCreatedKey = "order_created"
)

func NewRabbitMQBroker(client *rabbitmq.Client) domain.MessageBroker {
	// In a real scenario, you might want to wait for connection
	// For simplicity, we attempt to declare. If it fails due to connection,
	// the handleReconnect in client will eventually make it work if we retry.
	// But usually, it's better to declare it here or in an Init method.
	
	// We'll try to declare it. We use a background context or a timeout context.
	go func() {
		for {
			if client.IsConnected() {
				err := client.ExchangeDeclare(OrderExchange, "topic", true, false)
				if err == nil {
					log.Printf("Exchange %s declared successfully", OrderExchange)
					return
				}
				log.Printf("Failed to declare exchange: %v, retrying...", err)
			}
			time.Sleep(2 * time.Second)
		}
	}()

	return &rabbitMQBroker{
		client: client,
	}
}

// Add time import to the file
func (b *rabbitMQBroker) PublishOrderCreated(ctx context.Context, order *domain.Order) error {
	body, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshal order: %w", err)
	}

	msg := amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		Body:         body,
	}

	err = b.client.Publish(ctx, OrderExchange, OrderCreatedKey, msg)
	if err != nil {
		return fmt.Errorf("failed to publish order created event: %w", err)
	}

	return nil
}
