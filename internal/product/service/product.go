package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/qinyul/resilient-ecommerce-microservices/internal/product/domain"
)

type productService struct {
	repo domain.ProductRepository
}

func NewProductService(repo domain.ProductRepository) domain.ProductService {
	return &productService{
		repo: repo,
	}
}

func (s *productService) CreateProduct(ctx context.Context, input domain.CreateProductInput) (*domain.Product, error) {
	slog.InfoContext(ctx, "attempting to create product",
		"product_name", input.Name)

	p := &domain.Product{
		Name:        input.Name,
		Description: input.Description,
		Price:       input.Price,
		IsActive:    true,
	}

	if err := s.repo.CreateProduct(ctx, p); err != nil {
		slog.ErrorContext(ctx, "failed to create product", "product_name", input.Name, "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "product created successfully", "product_name", input.Name)
	return p, nil
}

func (s *productService) GetProduct(ctx context.Context, id string) (*domain.Product, error) {
	product, err := s.repo.GetProduct(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("productService.GetProduct: failed to retrieve product :%s %w", id, err)
	}
	return product, nil
}
