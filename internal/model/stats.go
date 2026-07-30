package model

type SalesStats struct {
	TotalSales  int64            `json:"total_sales"`
	Statuses    []StatusStat     `json:"statuses"`
	TopProducts []TopProductStat `json:"top_products"`
}

type StatusStat struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

type TopProductStat struct {
	ProductID int64  `json:"product_id"`
	Name      string `json:"name"`
	Sold      int64  `json:"sold"`
}
