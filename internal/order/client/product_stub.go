package client

import (
	"context"
	"fmt"

	"github.com/qinyul/resilient-ecommerce-microservices/internal/order/domain"
)

type productStubClient struct{}

func NewProductStubClient() domain.ProductClient {
	return &productStubClient{}
}

func (c *productStubClient) GetProduct(ctx context.Context, productID string) (*domain.Product, error) {
	// Stub implementation returning mock products
	// In a real scenario, this would call the Product API via gRPC or HTTP
	switch productID {
	case "550e8400-e29b-41d4-a716-446655440000":
		return &domain.Product{
			ID:   "550e8400-e29b-41d4-a716-446655440000",
			Name: "Product One",
			Price: domain.Money{
				Currency: "IDR",
				Units:    100000,
				Nanos:    0,
			},
		}, nil
	case "550e8400-e29b-41d4-a716-446655440001":
		return &domain.Product{
			ID:   "550e8400-e29b-41d4-a716-446655440001",
			Name: "Product Two",
			Price: domain.Money{
				Currency: "IDR",
				Units:    50000,
				Nanos:    0,
			},
		}, nil
	default:
		return nil, fmt.Errorf("product not found: %s", productID)
	}
}
