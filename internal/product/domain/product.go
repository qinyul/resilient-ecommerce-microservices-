package domain

import (
	"context"
	"errors"
	"time"

	"github.com/go-playground/validator/v10"
)

var (
	Validate           = validator.New()
	ErrProductNotFound = errors.New("product not found")
)

type Money struct {
	Currency string `json:"currency" validate:"required,len=3"`
	Units    int64  `json:"units"`
	Nanos    int32  `json:"nanos"`
}

type Product struct {
	ID          string    `json:"id"`
	Name        string    `json:"name" validate:"required"`
	Description string    `json:"description"`
	Price       Money     `json:"price" validate:"required"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ProductRepository interface {
	CreateProduct(ctx context.Context, product *Product) error
	GetProduct(ctx context.Context, id string) (*Product, error)
}

type CreateProductInput struct {
	Name        string `validate:"required"`
	Description string
	Price       Money  `validate:"required"`
}

type ProductService interface {
	CreateProduct(ctx context.Context, input CreateProductInput) (*Product, error)
	GetProduct(ctx context.Context, id string) (*Product, error)
}
