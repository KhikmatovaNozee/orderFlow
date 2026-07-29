package order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func (r *Repo) PlaceOrder(ctx context.Context, userID int64, items []model.OrderLineInput) (*model.Order, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var (
		orderID   int64
		createdAt time.Time
	)
	err = tx.QueryRow(ctx,
		`INSERT INTO orders (user_id, status, total) VALUES ($1, $2, 0)
		 RETURNING id, created_at`,
		userID, model.OrderStatusNew,
	).Scan(&orderID, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	var total int64
	for _, item := range items {
		price, err := decrementStock(ctx, tx, item)
		if err != nil {
			return nil, err
		}

		total += price * item.Quantity

		_, err = tx.Exec(ctx,
			`INSERT INTO order_items (order_id, product_id, quantity, price)
			 VALUES ($1, $2, $3, $4)`,
			orderID, item.ProductID, item.Quantity, price,
		)
		if err != nil {
			return nil, fmt.Errorf("insert order item: %w", err)
		}
	}

	if _, err := tx.Exec(ctx, `UPDATE orders SET total = $1 WHERE id = $2`, total, orderID); err != nil {
		return nil, fmt.Errorf("update order total: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &model.Order{
		ID:        orderID,
		UserID:    userID,
		Status:    model.OrderStatusNew,
		Total:     total,
		CreatedAt: createdAt,
	}, nil
}

func decrementStock(ctx context.Context, tx pgx.Tx, item model.OrderLineInput) (int64, error) {
	var (
		status string
		price  int64
	)
	err := tx.QueryRow(ctx,
		`SELECT status, price FROM products WHERE id = $1 FOR UPDATE`,
		item.ProductID,
	).Scan(&status, &price)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, model.ErrInvalid
		}
		return 0, fmt.Errorf("lock product %d: %w", item.ProductID, err)
	}
	if status != model.ProductStatusActive {
		return 0, model.ErrInvalid
	}

	tag, err := tx.Exec(ctx,
		`UPDATE products SET stock = stock - $1 WHERE id = $2 AND stock >= $1`,
		item.Quantity, item.ProductID,
	)
	if err != nil {
		return 0, fmt.Errorf("decrement stock for product %d: %w", item.ProductID, err)
	}
	if tag.RowsAffected() == 0 {
		return 0, model.ErrNoStock
	}

	return price, nil
}
