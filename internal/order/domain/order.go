package domain

import (
	"context"
	"errors"
	"time"

	"github.com/go-playground/validator/v10"
)

// OrderStatus defines the various states an order can be in.
type OrderStatus string

const (
	StatusPending   OrderStatus = "PENDING"
	StatusPaid      OrderStatus = "PAID"
	StatusCancelled OrderStatus = "CANCELLED"
	StatusCompleted OrderStatus = "COMPLETED"
)

var (
	ErrInvalidStatus = errors.New("invalid order status transition")
	ErrOrderNotFound = errors.New("order not found")
	Validate         = validator.New()
)

// Money represents a financial value with currency.
type Money struct {
	Currency string `json:"currency" validate:"required,len=3"`
	Units    int64  `json:"units"`
	Nanos    int32  `json:"nanos"`
}

type OrderItem struct {
	ID        string `json:"id"`
	OrderID   string `json:"order_id"`
	ProductID string `json:"product_id" validate:"required,uuid"`
	Quantity  int    `json:"quantity" validate:"gt=0"`
	UnitPrice Money  `json:"unit_price" validate:"required"`
}

// Order represents the core domain entity for an order in the system.
type Order struct {
	ID         string      `json:"id" validate:"required"`
	UserID     string      `json:"user_id" validate:"required"`
	Status     OrderStatus `json:"status" validate:"required,oneof=PENDING PAID CANCELLED COMPLETED"`
	TotalPrice Money       `json:"total_price"`
	Items      []OrderItem `json:"items" validate:"required"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

// Validate ensures the order data is consistent and valid using struct tags.
func (o *Order) Validate() error {
	return Validate.Struct(o)
}

// CanTransitionTo checks if the order can move from its current status to the new status.
func (o *Order) CanTransitionTo(newStatus OrderStatus) bool {
	switch o.Status {
	case StatusPending:
		return newStatus == StatusPaid || newStatus == StatusCancelled
	case StatusPaid:
		return newStatus == StatusCompleted || newStatus == StatusCancelled
	case StatusCancelled, StatusCompleted:
		return false // Terminal states
	default:
		return false
	}
}

// OrderRepository defines the interface for persisting and retrieving orders.
type OrderRepository interface {
	CreateOrder(ctx context.Context, order *Order) error
	GetOrder(ctx context.Context, id string) (*Order, error)
	UpdateStatus(ctx context.Context, id string, status OrderStatus) error
}

// MessageBroker defines the interface for publishing order-related events.
type MessageBroker interface {
	// PublishOrderCreated should return an error if the message fails to publish.
	// In a resilient system, this allows the caller to handle failures (e.g., retries or outbox).
	 PublishOrderCreated(ctx context.Context, order *Order) error
}

type CreateOrderItem struct {
	ProductID string `validate:"required"`
	Quantity  int    `validate:"gt=0"`
	UnitPrice Money  `validate:"required"`
}

type CreateOrderInput struct {
	UserID string            `validate:"required"`
	Items  []CreateOrderItem `validate:"required,min=1,dive"`
}

// OrderService defines the high-level business logic for order management.
type OrderService interface {
	PlaceOrder(ctx context.Context, input CreateOrderInput) (*Order, error)
	GetOrder(ctx context.Context, id string) (*Order, error)
	UpdateStatus(ctx context.Context, id string, status OrderStatus) error
}
