package model

import (
	"time"

	"github.com/google/uuid"
)

type Order struct {
	OrderId     uint64     `json:"order_id"`
	CustemerId  uuid.UUID  `json:"customer_id"`
	LineItems   []LineItem `json:"line_items"`
	OrderStatus string     `json:"status"`
	CreatedAt   *time.Time `json:"created_at"`
	ShippedAt   *time.Time `json:"shipped_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

type LineItem struct {
	ItemId   uuid.UUID `json:"item_id"`
	Quantity int       `json:"quantity"`
	Price    int       `json:"price"`
}
