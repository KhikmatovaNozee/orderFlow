package model

import "time"

type Product struct {
	ID        int64     `json:"id"`
	SellerID  int64     `json:"seller_id"`
	Name      string    `json:"name"`
	Price     int64     `json:"price"`
	Stock     int64     `json:"stock"`
	PhotoPath *string   `json:"photo_path"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
