package stats

import (
	"context"
	"fmt"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool: pool,
	}
}

func (r *Repository) Get(ctx context.Context, sellerID int64) (*model.SalesStats, error) {
	stats := &model.SalesStats{}
	err := r.pool.QueryRow(ctx, `SELECT COALESCE(SUM(o.total),0) FROM orders o JOIN order_items oi ON oi.order_id=o.id 
    JOIN products p ON p.id=oi.product_id WHERE p.seller_id=$1 AND o.status != 'cancelled'`, sellerID).Scan(&stats.TotalSales)

	if err != nil {
		return stats, fmt.Errorf("total sales: %w", err)
	}

	rows, err := r.pool.Query(ctx, `SELECT  o.status, COUNT(*) FROM orders o JOIN order_items oi ON oi.order_id=o.id 
    JOIN products p ON p.id=oi.product_id WHERE p.seller_id=$1 GROUP BY o.status`, sellerID)

	if err != nil {
		return stats, err
	}
	defer rows.Close()

	for rows.Next() {
		var s model.StatusStat
		if err := rows.Scan(&s.Status, &s.Count); err != nil {
			return stats, err
		}
		stats.Statuses = append(stats.Statuses, s)
	}

	rows, err = r.pool.Query(ctx, `
		SELECT p.id, p.name, SUM(oi.quantity) FROM order_items oi JOIN orders o ON o.id=oi.order_id JOIN products p ON p.id=oi.product_id
		WHERE p.seller_id=$1 AND o.status != 'cancelled' GROUP BY p.id,p.name ORDER BY SUM(oi.quantity) DESC LIMIT 10`, sellerID)

	if err != nil {
		return stats, err
	}
	defer rows.Close()

	for rows.Next() {
		var p model.TopProductStat
		if err := rows.Scan(&p.ProductID, &p.Name, &p.Sold); err != nil {
			return stats, err
		}
		stats.TopProducts = append(stats.TopProducts, p)
	}
	return stats, nil
}
