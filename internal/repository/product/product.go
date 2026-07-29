package product

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

type whereBuilder struct {
	conditions []string
	args       []any
}

func (b *whereBuilder) add(clause string, value any) {
	b.args = append(b.args, value)
	b.conditions = append(b.conditions, fmt.Sprintf(clause, len(b.args)))
}

func (b *whereBuilder) sql() string {
	if len(b.conditions) == 0 {
		return ""
	}
	return "WHERE " + strings.Join(b.conditions, " AND ")
}

func (r *Repo) List(ctx context.Context, f model.ProductFilter) (model.ProductListResult, error) {
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

	b := &whereBuilder{}
	if f.SellerID != nil {
		b.add("seller_id = $%d", *f.SellerID)
	}
	if f.Status != nil {
		b.add("status = $%d", *f.Status)
	}
	if f.Category != nil && *f.Category != "" {
		b.add("category = $%d", *f.Category)
	}
	if f.Query != nil && *f.Query != "" {
		b.add("name ILIKE $%d", "%"+*f.Query+"%")
	}
	if f.PriceMin != nil {
		b.add("price >= $%d", *f.PriceMin)
	}
	if f.PriceMax != nil {
		b.add("price <= $%d", *f.PriceMax)
	}

	var total int
	countQuery := fmt.Sprintf("SELECT count(*) FROM products %s", b.sql())
	if err := r.pool.QueryRow(ctx, countQuery, b.args...).Scan(&total); err != nil {
		return model.ProductListResult{}, fmt.Errorf("count products: %w", err)
	}

	offset := (page - 1) * limit
	listQuery := fmt.Sprintf(
		`SELECT id, seller_id, name, category, price, stock, status, photo_path, created_at, updated_at
		 FROM products %s
		 ORDER BY id
		 LIMIT $%d OFFSET $%d`,
		b.sql(), len(b.args)+1, len(b.args)+2,
	)
	args := append(append([]any{}, b.args...), limit, offset)

	rows, err := r.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return model.ProductListResult{}, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	items := make([]model.Product, 0, limit)
	for rows.Next() {
		var p model.Product
		if err := rows.Scan(&p.ID, &p.SellerID, &p.Name, &p.Category, &p.Price,
			&p.Stock, &p.Status, &p.PhotoPath, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return model.ProductListResult{}, fmt.Errorf("scan product: %w", err)
		}
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return model.ProductListResult{}, fmt.Errorf("iterate products: %w", err)
	}

	return model.ProductListResult{Items: items, Total: total, Page: page, Limit: limit}, nil
}

func (r *Repo) GetByID(ctx context.Context, id int64) (*model.Product, error) {
	const q = `SELECT id, seller_id, name, category, price, stock, status, photo_path, created_at, updated_at
	           FROM products WHERE id = $1`

	var p model.Product
	err := r.pool.QueryRow(ctx, q, id).Scan(&p.ID, &p.SellerID, &p.Name, &p.Category,
		&p.Price, &p.Stock, &p.Status, &p.PhotoPath, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("get product: %w", err)
	}
	return &p, nil
}

func (r *Repo) Create(ctx context.Context, p model.Product) (model.Product, error) {
	const q = `INSERT INTO products (seller_id, name, category, price, stock, status, photo_path)
	           VALUES ($1, $2, $3, $4, $5, $6, $7)
	           RETURNING id, created_at, updated_at`

	err := r.pool.QueryRow(ctx, q,
		p.SellerID, p.Name, p.Category, p.Price, p.Stock, p.Status, p.PhotoPath,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return model.Product{}, fmt.Errorf("create product: %w", err)
	}
	return p, nil
}

func (r *Repo) Update(ctx context.Context, p model.Product) (model.Product, error) {
	const q = `UPDATE products
	           SET name = $1, category = $2, price = $3, stock = $4, status = $5, photo_path = $6, updated_at = now()
	           WHERE id = $7
	           RETURNING updated_at`

	err := r.pool.QueryRow(ctx, q,
		p.Name, p.Category, p.Price, p.Stock, p.Status, p.PhotoPath, p.ID,
	).Scan(&p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Product{}, model.ErrNotFound
		}
		return model.Product{}, fmt.Errorf("update product: %w", err)
	}
	return p, nil
}
