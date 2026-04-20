package domain

import (
	"context"
)

type Product struct {
	ID    string
	Name  string
	Price Money
}

type ProductClient interface {
	GetProduct(ctx context.Context, productID string) (*Product, error)
}
