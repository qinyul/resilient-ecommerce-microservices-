package domain

import (
	"context"
	"github.com/qinyul/resilient-ecommerce-microservices/internal/order/domain"
)

// PaymentOrderEvent is the event received when an order is created.
type PaymentOrderEvent struct {
	ID         string       `json:"id"`
	TotalPrice domain.Money `json:"total_price"`
}

// PaymentCompletedEvent is the event published when a payment is successful.
type PaymentCompletedEvent struct {
	OrderID string `json:"order_id"`
}

// PaymentService defines the business logic for payment processing.
type PaymentService interface {
	ProcessPayment(ctx context.Context, order PaymentOrderEvent) error
}
