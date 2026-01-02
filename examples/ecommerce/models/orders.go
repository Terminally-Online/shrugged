package models

import "time"

type OrdersExtension struct {
	Items *OrderItemsList `json:"items,omitempty"`
}

type Orders struct {
	ID                int64      `json:"id"`
	CustomerID        int64      `json:"customer_id"`
	ShippingAddressID *int64     `json:"shipping_address_id,omitempty"`
	BillingAddressID  *int64     `json:"billing_address_id,omitempty"`
	SubtotalCents     int64      `json:"subtotal_cents"`
	TaxCents          int64      `json:"tax_cents"`
	ShippingCents     int64      `json:"shipping_cents"`
	TotalCents        int64      `json:"total_cents"`
	Notes             *string    `json:"notes,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         *time.Time `json:"updated_at,omitempty"`
	OrdersExtension
}
