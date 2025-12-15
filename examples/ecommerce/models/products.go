package models

import (
	"encoding/json"
	"time"
)

type Products struct {
	ID              int64           `json:"id"`
	CategoryID      *int64          `json:"category_id"`
	Sku             string          `json:"sku"`
	Name            string          `json:"name"`
	Description     *string         `json:"description"`
	PriceCents      int64           `json:"price_cents"`
	QuantityInStock int32           `json:"quantity_in_stock"`
	WeightGrams     *int32          `json:"weight_grams"`
	IsActive        bool            `json:"is_active"`
	Metadata        json.RawMessage `json:"metadata"`
	Tags            []string        `json:"tags"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       *time.Time      `json:"updated_at"`
}
