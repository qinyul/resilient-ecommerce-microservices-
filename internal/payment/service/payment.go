package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/qinyul/resilient-ecommerce-microservices/internal/payment/domain"
	"github.com/qinyul/resilient-ecommerce-microservices/pkg/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

type paymentService struct {
	rmqClient *rabbitmq.Client
}

func NewPaymentService(rmqClient *rabbitmq.Client) domain.PaymentService {
	return &paymentService{
		rmqClient: rmqClient,
	}
}

func (s *paymentService) ProcessPayment(ctx context.Context, order domain.PaymentOrderEvent) error {
	slog.Info("Processing payment for order", 
		"order_id", order.ID, 
		"amount", order.TotalPrice.Units, 
		"currency", order.TotalPrice.Currency)

	// Simulate processing time
	time.Sleep(500 * time.Millisecond)

	slog.Info("Payment successful for order", "order_id", order.ID)

	// Publish PaymentCompletedEvent
	event := domain.PaymentCompletedEvent{
		OrderID: order.ID,
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal payment completed event: %w", err)
	}

	err = s.rmqClient.Publish(ctx, "order_exchange", "payment_completed", amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
	if err != nil {
		return fmt.Errorf("failed to publish payment completed event: %w", err)
	}

	return nil
}
