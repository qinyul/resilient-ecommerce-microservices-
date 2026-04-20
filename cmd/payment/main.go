package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/qinyul/resilient-ecommerce-microservices/internal/order/domain"
	"github.com/qinyul/resilient-ecommerce-microservices/pkg/broker"
	"github.com/qinyul/resilient-ecommerce-microservices/pkg/config"
	"github.com/qinyul/resilient-ecommerce-microservices/pkg/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

type PaymentOrderEvent struct {
	ID         string       `json:"id"`
	TotalPrice domain.Money `json:"total_price"`
}

type PaymentCompletedEvent struct {
	OrderID string `json:"order_id"`
}

func main() {
	// 1. Initialize Structured Logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 2. Load Configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// 3. Setup Graceful Shutdown Context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("Starting Payment Service...")

	// 4. Initialize RabbitMQ Client
	rabbitMQURL := fmt.Sprintf("amqp://%s:%s@%s:%s/",
		cfg.RabbitMQ.User, cfg.RabbitMQ.Password, cfg.RabbitMQ.Host, cfg.RabbitMQ.Port)
	
	rmqClient := rabbitmq.NewClient(rabbitMQURL)
	defer rmqClient.Close()

	// 5. Define Handler
	handler := func(ctx context.Context, d amqp.Delivery) (error, bool) {
		slog.Info("Received an event", "delivery_tag", d.DeliveryTag)

		var order PaymentOrderEvent
		if err := json.Unmarshal(d.Body, &order); err != nil {
			slog.Error("Error decoding json", "error", err)
			return nil, false // Don't requeue malformed JSON
		}

		// --- BUSINESS LOGIC ---
		slog.Info("Processing payment", 
			"order_id", order.ID, 
			"amount", order.TotalPrice.Units, 
			"currency", order.TotalPrice.Currency)

		time.Sleep(500 * time.Millisecond)
		// ----------------------

		slog.Info("Payment successful", "order_id", order.ID)

		// Publish Payment Completed Event
		paymentEvent := PaymentCompletedEvent{
			OrderID: order.ID,
		}
		body, _ := json.Marshal(paymentEvent)
		err := rmqClient.Publish(ctx, "order_exchange", "payment_completed", amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
		if err != nil {
			slog.Error("Failed to publish payment completed event", "error", err)
			return err, true // Requeue if publication fails
		}

		return nil, false
	}

	// 6. Initialize and Start Consumer
	consumer := broker.NewEventConsumer(
		rmqClient,
		"payment_order_created",
		"order_exchange",
		"order_created",
		"payment_service",
		handler,
	)

	go consumer.Start(ctx)

	// Wait for termination signal
	<-ctx.Done()
	slog.Info("Shutting down Payment Service...")
}
