package models

type OrderItems struct {
	ID             int64 `json:"id"`
	OrderID        int64 `json:"order_id"`
	ProductID      int64 `json:"product_id"`
	Quantity       int32 `json:"quantity"`
	UnitPriceCents int64 `json:"unit_price_cents"`
	TotalCents     int64 `json:"total_cents"`
}
