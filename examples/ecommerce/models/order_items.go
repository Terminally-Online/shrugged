package models

import (
	"encoding/json"
	"fmt"
)

type OrderItemsExtension struct{}

type OrderItems struct {
	OrderItemsExtension
	ID             int64 `json:"id"`
	OrderID        int64 `json:"order_id"`
	ProductID      int64 `json:"product_id"`
	Quantity       int32 `json:"quantity"`
	UnitPriceCents int64 `json:"unit_price_cents"`
	TotalCents     int64 `json:"total_cents"`
}

type OrderItemsList []OrderItems

func (o *OrderItemsList) Scan(src any) error {
	switch v := src.(type) {
	case []byte:
		if len(v) == 0 {
			return nil
		}
		return json.Unmarshal(v, o)
	case string:
		if v == "" {
			return nil
		}
		return json.Unmarshal([]byte(v), o)
	case nil:
		return nil
	default:
		return fmt.Errorf("cannot scan %T into OrderItemsList", src)
	}
}
