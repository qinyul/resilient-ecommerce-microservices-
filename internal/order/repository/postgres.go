package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/qinyul/resilient-ecommerce-microservices/internal/order/domain"
)

type postgresOrderRepository struct {
	db *sql.DB
}

// NewPostgresOrderRepository creates a new instance of the repository.
func NewPostgresOrderRepository(db *sql.DB) domain.OrderRepository {
	return &postgresOrderRepository{
		db: db,
	}
}

// CreateOrder inserts a new order into the database.
func (r *postgresOrderRepository) CreateOrder(ctx context.Context, order *domain.Order) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	const query = `
		INSERT INTO orders (user_id, status, currency, total_units, total_nanos, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	err = tx.QueryRowContext(
		ctx,
		query,
		order.UserID,
		order.Status,
		order.TotalPrice.Currency,
		order.TotalPrice.Units,
		order.TotalPrice.Nanos,
		order.CreatedAt,
		order.UpdatedAt,
	).Scan(&order.ID)

	if err != nil {
		return fmt.Errorf("failed to insert order: %w", err)
	}

	// insert items
	const itemQuery = `
		INSERT INTO order_items (order_id, product_id, quantity, currency, unit_units, unit_nanos)
		VALUES($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	for i := range order.Items {
		err = tx.QueryRowContext(
			ctx,
			itemQuery,
			order.ID,
			order.Items[i].ProductID,
			order.Items[i].Quantity,
			order.Items[i].UnitPrice.Currency,
			order.Items[i].UnitPrice.Units,
			order.Items[i].UnitPrice.Nanos,
		).Scan(&order.Items[i].ID)

		if err != nil {
			return fmt.Errorf("failed to insert order item at index %d: %w", i, err)
		}
		order.Items[i].OrderID = order.ID
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetOrder retrieves an order by its ID.
func (r *postgresOrderRepository) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	return r.getOrder(ctx, r.db, id, false)
}

// ListOrders retrieves all orders for a spesific user.
func (r *postgresOrderRepository) ListOrders(ctx context.Context, userID string) ([]*domain.Order, error) {
	query := `
		SELECT
			o.id, o.user_id, o.status, o.currency, o.total_units, o.total_nanos, o.created_at, o.updated_at,
			oi.id,oi.product_id, oi.quantity, oi.currency, oi.unit_units, oi.unit_nanos
		FROM orders o
		LEFT JOIN order_items oi ON o.id = oi.order_id
		WHERE o.user_id = $1
		ORDER BY o.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list orders: %w", err)
	}
	defer rows.Close()

	orderMap := make(map[string]*domain.Order)
	var orderIDs []string

	for rows.Next() {
		var itemID, itemProductID sql.NullString
		var itemQuantity sql.NullInt32
		var itemCurrency sql.NullString
		var itemUnits sql.NullInt64
		var itemNanos sql.NullInt32

		var o domain.Order
		err := rows.Scan(
			&o.ID,
			&o.UserID,
			&o.Status,
			&o.TotalPrice.Currency,
			&o.TotalPrice.Units,
			&o.TotalPrice.Nanos,
			&o.CreatedAt,
			&o.UpdatedAt,
			&itemID,
			&itemProductID,
			&itemQuantity,
			&itemCurrency,
			&itemUnits,
			&itemNanos,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order row: %w", err)
		}

		if _, ok := orderMap[o.ID]; !ok {
			order := o
			order.Items = []domain.OrderItem{}
			orderMap[o.ID] = &order
			orderIDs = append(orderIDs, o.ID)
		}

		if itemID.Valid {
			orderMap[o.ID].Items = append(orderMap[o.ID].Items, domain.OrderItem{
				ID:        itemID.String,
				OrderID:   o.ID,
				ProductID: itemProductID.String,
				Quantity:  int(itemQuantity.Int32),
				UnitPrice: domain.Money{
					Currency: itemCurrency.String,
					Units:    itemUnits.Int64,
					Nanos:    itemNanos.Int32,
				},
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating order rows: %w", err)
	}

	orders := make([]*domain.Order, 0, len(orderIDs))
	for _, id := range orderIDs {
		orders = append(orders, orderMap[id])
	}

	return orders, nil
}

func (r *postgresOrderRepository) getOrder(ctx context.Context, q interface {
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
}, id string, forUpdate bool) (*domain.Order, error) {
	query := `
		SELECT
			o.id, o.user_id, o.status, o.currency, o.total_units, o.total_nanos, o.created_at, o.updated_at,
			oi.id, oi.product_id, oi.quantity, oi.currency, oi.unit_units, oi.unit_nanos
		FROM orders o
		LEFT JOIN order_items oi ON o.id = oi.order_id
		WHERE o.id = $1
	`
	if forUpdate {
		query += " FOR UPDATE OF o"
	}

	rows, err := q.QueryContext(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order: %w", err)
	}
	defer rows.Close()

	var order *domain.Order
	for rows.Next() {
		var itemID, itemProductID sql.NullString
		var itemQuantity sql.NullInt32
		var itemCurrency sql.NullString
		var itemUnits sql.NullInt64
		var itemNanos sql.NullInt32

		var o domain.Order
		err := rows.Scan(
			&o.ID,
			&o.UserID,
			&o.Status,
			&o.TotalPrice.Currency,
			&o.TotalPrice.Units,
			&o.TotalPrice.Nanos,
			&o.CreatedAt,
			&o.UpdatedAt,
			&itemID,
			&itemProductID,
			&itemQuantity,
			&itemCurrency,
			&itemUnits,
			&itemNanos,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order row: %w", err)
		}

		if order == nil {
			order = &o
			order.Items = []domain.OrderItem{}
		}

		if itemID.Valid {
			order.Items = append(order.Items, domain.OrderItem{
				ID:        itemID.String,
				OrderID:   order.ID,
				ProductID: itemProductID.String,
				Quantity:  int(itemQuantity.Int32),
				UnitPrice: domain.Money{
					Currency: itemCurrency.String,
					Units:    itemUnits.Int64,
					Nanos:    itemNanos.Int32,
				},
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating order rows: %w", err)
	}

	if order == nil {
		return nil, fmt.Errorf("%w: %s", domain.ErrOrderNotFound, id)
	}

	return order, nil
}

// UpdateStatus changes the status of an existing order within a transaction.
func (r *postgresOrderRepository) UpdateStatus(ctx context.Context, id string, status domain.OrderStatus) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Get with lock
	order, err := r.getOrder(ctx, tx, id, true)
	if err != nil {
		return err
	}

	// 2. Validate transition (domain logic should ideally be here if it depends on current state)
	if !order.CanTransitionTo(status) {
		return fmt.Errorf("%w from %s to %s", domain.ErrInvalidStatus, order.Status, status)
	}

	// 3. Update
	const query = `
		UPDATE orders
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`
	_, err = tx.ExecContext(ctx, query, string(status), id)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	return tx.Commit()
}
