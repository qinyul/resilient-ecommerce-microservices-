package handler

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/qinyul/resilient-ecommerce-microservices/internal/payment/domain"
	amqp "github.com/rabbitmq/amqp091-go"
)

type PaymentHandler struct {
	service domain.PaymentService
}

func NewPaymentHandler(service domain.PaymentService) *PaymentHandler {
	return &PaymentHandler{
		service: service,
	}
}

func (h *PaymentHandler) HandleOrderCreated(ctx context.Context, d amqp.Delivery) (error, bool) {
	slog.Info("Received an order created event", "delivery_tag", d.DeliveryTag)

	var order domain.PaymentOrderEvent
	if err := json.Unmarshal(d.Body, &order); err != nil {
		slog.Error("Fatal: Error decoding json (Dead Letter)", "error", err)
		return err, false // Fatal: Don't requeue malformed JSON
	}

	if err := h.service.ProcessPayment(ctx, order); err != nil {
		slog.Error("Transient: Payment processing failed (Requeue)", "error", err)
		return err, true // Transient: Requeue for retry
	}

	return nil, false
}
