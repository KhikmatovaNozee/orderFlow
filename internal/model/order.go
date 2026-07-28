package model

import "time"

const (
	OrderStatusNew       = "new"
	OrderStatusPaid      = "paid"
	OrderStatusShipped   = "shipped"
	OrderStatusCancelled = "cancelled"
)

type Order struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Status    string    `json:"status"`
	Total     int64     `json:"total"`
	CreatedAt time.Time `json:"created_at"`
}
