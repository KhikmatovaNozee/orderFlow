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
	Tracking  string    `json:"tracking"`
	CreatedAt time.Time `json:"created_at"`
}

type OrderFilter struct {
	UserID *int64
	Status *string
	Page   int
	Limit  int
}

type OrderListResult struct {
	Items []Order
	Total int
	Page  int
	Limit int
}

type OrderDetail struct {
	Order
	Items []OrderItem `json:"items"`
}
