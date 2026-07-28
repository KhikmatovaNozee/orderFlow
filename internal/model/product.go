package model

import "time"

const (
	ProductStatusActive = "active"
	ProductStatusHidden = "hidden"
)

type Product struct {
	ID        int64     `json:"id"`
	SellerID  int64     `json:"seller_id"`
	Name      string    `json:"name"`
	Category  string    `json:"category"`
	Price     int64     `json:"price"`
	Stock     int64     `json:"stock"`
	Status    string    `json:"status"`
	PhotoPath *string   `json:"photo_path"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ProductFilter struct {
	SellerID *int64
	Status   *string
	Query    *string
	Category *string
	PriceMin *int64
	PriceMax *int64
	Page     int
	Limit    int
}

type ProductListResult struct {
	Items []Product
	Total int
	Page  int
	Limit int
}
