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

const (
	defaultLimit = 20
	maxLimit     = 100
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

func (r *Repo) List(ctx context.Context, f model.OrderFilter) (model.OrderListResult, error) {
	page := f.Page
	if page < 1 {
		page = 1
	}
	limit := f.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	conditions := []string{"user_id = $1"}
	args := []any{*f.UserID}

	if f.Status != nil {
		args = append(args, *f.Status)
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
	}

	where := "WHERE " + conditions[0]
	for _, c := range conditions[1:] {
		where += " AND " + c
	}

	var total int
	countQuery := fmt.Sprintf("SELECT count(*) FROM orders %s", where)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return model.OrderListResult{}, fmt.Errorf("count orders: %w", err)
	}

	offset := (page - 1) * limit
	listArgs := append(append([]any{}, args...), limit, offset)
	listQuery := fmt.Sprintf(
		`SELECT id, user_id, status, total, created_at
		 FROM orders %s
		 ORDER BY id DESC
		 LIMIT $%d OFFSET $%d`,
		where, len(args)+1, len(args)+2,
	)

	rows, err := r.pool.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return model.OrderListResult{}, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	items := make([]model.Order, 0, limit)
	for rows.Next() {
		var o model.Order
		if err := rows.Scan(&o.ID, &o.UserID, &o.Status, &o.Total, &o.CreatedAt); err != nil {
			return model.OrderListResult{}, fmt.Errorf("scan order: %w", err)
		}
		items = append(items, o)
	}
	if err := rows.Err(); err != nil {
		return model.OrderListResult{}, fmt.Errorf("iterate orders: %w", err)
	}

	return model.OrderListResult{Items: items, Total: total, Page: page, Limit: limit}, nil
}

func (r *Repo) GetDetail(ctx context.Context, id int64) (*model.OrderDetail, error) {
	var detail model.OrderDetail

	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, status, total, created_at FROM orders WHERE id = $1`,
		id,
	).Scan(&detail.ID, &detail.UserID, &detail.Status, &detail.Total, &detail.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("get order: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, order_id, product_id, quantity, price FROM order_items WHERE order_id = $1 ORDER BY id`,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("list order items: %w", err)
	}
	defer rows.Close()

	detail.Items = make([]model.OrderItem, 0)
	for rows.Next() {
		var it model.OrderItem
		if err := rows.Scan(&it.ID, &it.OrderID, &it.ProductID, &it.Quantity, &it.Price); err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}
		detail.Items = append(detail.Items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate order items: %w", err)
	}

	return &detail, nil
}

func (r *Repo) ListSellerOrders(ctx context.Context, sellerID int64, f model.OrderFilter) (model.OrderListResult, error) {
	args := []any{sellerID}
	query := `SELECT DISTINCT o.id, o.user_id, o.status, o.total, o.created_at FROM orders o JOIN order_items oi ON oi.order_id=o.id
			JOIN products p ON p.id=oi.product_id WHERE p.seller_id=$1`

	if f.Status != nil {
		query += " AND o.status=$2"
		args = append(args, *f.Status)
	}

	query += " ORDER BY o.id DESC"
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return model.OrderListResult{}, err
	}
	defer rows.Close()

	items := make([]model.Order, 0)
	for rows.Next() {
		var o model.Order
		err := rows.Scan(&o.ID, &o.UserID, &o.Status, &o.Total, &o.CreatedAt)
		if err != nil {
			return model.OrderListResult{}, err
		}
		items = append(items, o)
	}

	return model.OrderListResult{
		Items: items,
		Total: len(items),
	}, nil
}

func (r *Repo) Ship(ctx context.Context, id int64) (*model.Order, error) {
	var o model.Order
	err := r.pool.QueryRow(ctx,
		`UPDATE orders SET status = $1 WHERE id = $2 AND status = $3
		 RETURNING id, user_id, status, total, created_at`,
		model.OrderStatusShipped, id, model.OrderStatusPaid,
	).Scan(&o.ID, &o.UserID, &o.Status, &o.Total, &o.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrInvalid
		}
		return nil, fmt.Errorf("ship order: %w", err)
	}
	return &o, nil
}

func (r *Repo) GetSellerOrder(ctx context.Context, sellerID int64, orderID int64) (*model.OrderDetail, error) {
	const q = `SELECT o.id, o.user_id, o.status, o.total, o.created_at FROM orders o JOIN order_items oi ON oi.order_id = o.id
    JOIN products p ON p.id = oi.product_id WHERE o.id=$1 AND p.seller_id=$2`

	var detail model.OrderDetail
	err := r.pool.QueryRow(ctx, q, orderID, sellerID).Scan(&detail.ID, &detail.UserID, &detail.Status, &detail.Total, &detail.CreatedAt)
	if err != nil {
		return nil, model.ErrNotFound
	}
	return &detail, nil
}

func (r *Repo) Pay(ctx context.Context, id int64) (*model.Order, error) {
	var o model.Order
	err := r.pool.QueryRow(ctx,
		`UPDATE orders SET status = $1 WHERE id = $2 AND status = $3
		 RETURNING id, user_id, status, total, created_at`,
		model.OrderStatusPaid, id, model.OrderStatusNew,
	).Scan(&o.ID, &o.UserID, &o.Status, &o.Total, &o.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrInvalid
		}
		return nil, fmt.Errorf("pay order: %w", err)
	}
	return &o, nil
}

func (r *Repo) Cancel(ctx context.Context, id int64) (*model.Order, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var o model.Order
	err = tx.QueryRow(ctx,
		`UPDATE orders SET status = $1 WHERE id = $2 AND status = $3
		 RETURNING id, user_id, status, total, created_at`,
		model.OrderStatusCancelled, id, model.OrderStatusNew,
	).Scan(&o.ID, &o.UserID, &o.Status, &o.Total, &o.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrInvalid
		}
		return nil, fmt.Errorf("cancel order: %w", err)
	}

	rows, err := tx.Query(ctx,
		`SELECT product_id, quantity FROM order_items WHERE order_id = $1`, id,
	)
	if err != nil {
		return nil, fmt.Errorf("list order items for cancel: %w", err)
	}

	type line struct {
		productID int64
		quantity  int64
	}
	var lines []line
	for rows.Next() {
		var l line
		if err := rows.Scan(&l.productID, &l.quantity); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan order item for cancel: %w", err)
		}
		lines = append(lines, l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate order items for cancel: %w", err)
	}

	for _, l := range lines {
		if _, err := tx.Exec(ctx,
			`UPDATE products SET stock = stock + $1 WHERE id = $2`,
			l.quantity, l.productID,
		); err != nil {
			return nil, fmt.Errorf("return stock for product %d: %w", l.productID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit cancel tx: %w", err)
	}

	return &o, nil
}
