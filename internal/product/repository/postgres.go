package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/qinyul/resilient-ecommerce-microservices/internal/product/domain"
)

type postgresProductRepository struct {
	db *sql.DB
}

func NewPostgresProductRepository(db *sql.DB) domain.ProductRepository {
	return &postgresProductRepository{
		db: db,
	}
}

func (r *postgresProductRepository) CreateProduct(ctx context.Context, p *domain.Product) error {
	const query = `
		INSERT INTO products (name, description, currency, price_units, price_nanos, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now

	err := r.db.QueryRowContext(
		ctx,
		query,
		p.Name,
		p.Description,
		p.Price.Currency,
		p.Price.Units,
		p.Price.Nanos,
		p.CreatedAt,
		p.UpdatedAt,
	).Scan(&p.ID)

	if err != nil {
		return fmt.Errorf("failed to insert product: %w", err)
	}

	return nil
}

func (r *postgresProductRepository) GetProduct(ctx context.Context, id string) (*domain.Product, error) {
	const query = `
		SELECT id, name, description, currency, price_units, price_nanos, created_at, updated_at
		FROM products
		WHERE id = $1
	`

	var p domain.Product
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID,
		&p.Name,
		&p.Description,
		&p.Price.Currency,
		&p.Price.Units,
		&p.Price.Nanos,
		&p.CreatedAt,
		&p.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrProductNotFound
		}
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	return &p, nil
}
