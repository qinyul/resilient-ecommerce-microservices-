package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/qinyul/resilient-ecommerce-microservices/internal/order/domain"
)

type orderService struct {
	repo          domain.OrderRepository
	broker        domain.MessageBroker
	productClient domain.ProductClient
}

func NewOrderService(
	repo domain.OrderRepository,
	broker domain.MessageBroker,
	productClient domain.ProductClient,
) domain.OrderService {
	return &orderService{
		repo:          repo,
		broker:        broker,
		productClient: productClient,
	}
}

func (s *orderService) PlaceOrder(ctx context.Context, input domain.CreateOrderInput) (*domain.Order, error) {
	now := time.Now()

	// Fallback to IDR as the standard currency
	orderCurrency := "IDR"

	var totalUnits int64
	var totalNanos int32

	items := make([]domain.OrderItem, len(input.Items))
	for i, item := range input.Items {
		// Fetch actual price from Product API
		product, err := s.productClient.GetProduct(ctx, item.ProductID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch product %s: %w", item.ProductID, err)
		}

		items[i] = domain.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: product.Price,
		}

		// Calculate item total: price * quantity
		itemTotalNanos := (product.Price.Units*1e9 + int64(product.Price.Nanos)) * int64(item.Quantity)

		totalUnits += itemTotalNanos / 1e9
		totalNanos += int32(itemTotalNanos % 1e9)

		if totalNanos >= 1e9 {
			totalUnits += int64(totalNanos / 1e9)
			totalNanos = totalNanos % 1e9
		}
	}

	order := &domain.Order{
		ID:     uuid.NewString(),
		UserID: input.UserID,
		Status: domain.StatusPending,
		TotalPrice: domain.Money{
			Currency: orderCurrency,
			Units:    totalUnits,
			Nanos:    totalNanos,
		},
		Items:     items,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.CreateOrder(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to save order to database: %w", err)
	}

	if err := s.broker.PublishOrderCreated(ctx, order); err != nil {
		return nil, fmt.Errorf("order saved but failed to publish event: %w", err)
	}

	return order, nil
}

func (s *orderService) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	order, err := s.repo.GetOrder(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	return order, nil
}

func (s *orderService) UpdateStatus(ctx context.Context, id string, status domain.OrderStatus) error {
	order, err := s.repo.GetOrder(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find order for status update: %w", err)
	}
	if !order.CanTransitionTo(status) {
		return fmt.Errorf("%w from %s to %s", domain.ErrInvalidStatus, order.Status, status)
	}
	if err := s.repo.UpdateStatus(ctx, id, status); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}
	return nil
}
