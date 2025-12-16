package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type ProductsExtension struct {
	Metadata *ProductMetadata `json:"metadata,omitempty"`
}

type Products struct {
	ProductsExtension
	ID              int64      `json:"id"`
	CategoryID      *int64     `json:"category_id,omitempty"`
	Sku             string     `json:"sku"`
	Name            string     `json:"name"`
	Description     *string    `json:"description,omitempty"`
	PriceCents      int64      `json:"price_cents"`
	QuantityInStock int32      `json:"quantity_in_stock"`
	WeightGrams     *int32     `json:"weight_grams,omitempty"`
	IsActive        bool       `json:"is_active"`
	Tags            []string   `json:"tags,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
}

type ProductMetadata struct {
	Dimensions map[string]string `json:"dimensions,omitempty"`
	Materials  []string          `json:"materials,omitempty"`
	Extra      map[string]string `json:"extra,omitempty"`
}

func (p *ProductMetadata) Scan(src any) error {
	var data map[string]any

	switch v := src.(type) {
	case []byte:
		if err := json.Unmarshal(v, &data); err != nil {
			return err
		}
	case string:
		if err := json.Unmarshal([]byte(v), &data); err != nil {
			return err
		}
	case nil:
		return nil
	default:
		return fmt.Errorf("cannot scan %T into ProductMetadata", src)
	}

	p.Dimensions = make(map[string]string)
	p.Extra = make(map[string]string)

	for k, v := range data {
		strVal := fmt.Sprintf("%v", v)

		if after, ok := strings.CutPrefix(k, "dimension:"); ok {
			p.Dimensions[after] = strVal
		} else if k == "materials" {
			if arr, ok := v.([]any); ok {
				for _, m := range arr {
					p.Materials = append(p.Materials, fmt.Sprintf("%v", m))
				}
			}
		} else {
			p.Extra[k] = strVal
		}
	}
	return nil
}

func (p ProductMetadata) Value() (driver.Value, error) {
	combined := make(map[string]any)
	for k, v := range p.Dimensions {
		combined["dimension:"+k] = v
	}
	if len(p.Materials) > 0 {
		combined["materials"] = p.Materials
	}
	for k, v := range p.Extra {
		combined[k] = v
	}
	return json.Marshal(combined)
}
