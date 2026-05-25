package domain

import (
	"context"
)

// Product represents a product in the system.
type Product struct {
	ID    string
	Name  string
	Price Money
}

// CreateProductInput represents the data needed to create a product.
type CreateProductInput struct {
	Name  string
	Price Money
}

// ProductClient defines the interface for fetching and creating product information.
type ProductClient interface {
	GetProduct(ctx context.Context, productID string) (*Product, error)
	CreateProduct(ctx context.Context, input CreateProductInput) (*Product, error)
}
