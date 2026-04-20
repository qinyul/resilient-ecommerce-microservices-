package handler

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/qinyul/resilient-ecommerce-microservices/internal/order/domain"
	amqp "github.com/rabbitmq/amqp091-go"
)

type PaymentCompletedHandler struct {
	orderService domain.OrderService
}

func NewPaymentCompletedHandler(orderService domain.OrderService) *PaymentCompletedHandler {
	return &PaymentCompletedHandler{
		orderService: orderService,
	}
}

func (h *PaymentCompletedHandler) HandlePaymentCompleted(ctx context.Context, d amqp.Delivery) (error, bool) {
	var event struct {
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal(d.Body, &event); err != nil {
		slog.Error("Fatal: Failed to unmarshal payment event (Dead Letter)", "error", err)
		return err, false // Fatal: Don't requeue malformed JSON
	}

	slog.Info("Received payment completed event", "order_id", event.OrderID)

	// Update order status to PAID
	err := h.orderService.UpdateStatus(ctx, event.OrderID, domain.StatusPaid)
	if err != nil {
		slog.Error("Transient: Failed to update order status (Requeue)", "order_id", event.OrderID, "error", err)
		return err, true // Transient: Requeue on failure
	}

	slog.Info("Order status updated to PAID", "order_id", event.OrderID)
	return nil, false
}
