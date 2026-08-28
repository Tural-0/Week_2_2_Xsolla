package models

import "time"

// Item represents a product available for purchase.
type Item struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       int       `json:"price"` // Price in cents
	Stock       int       `json:"stock"`
	CreatedAt   time.Time `json:"created_at"`
}

// LineItem is a line item in an order.
type LineItem struct {
	ItemID   int `json:"item_id"`
	Quantity int `json:"quantity"`
	Price    int `json:"price"` // Price at time of adding
}

// Order represents a completed purchase.
type Order struct {
	ID     int        `json:"id"`
	UserID int        `json:"user_id"`
	Items  []LineItem `json:"line_items"`
	Total  int        `json:"total"`
	Status string     `json:"status"` // pending, paid, failed
}

// Cart is a single cart row representing one item in a user's cart.
type Cart struct {
	UserID   int `json:"user_id"`
	ItemID   int `json:"item_id"`
	Quantity int `json:"quantity"`
}

// CartItem is an item in a user's cart joined with its product details.
type CartItem struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       int       `json:"price"` // Price in cents
	Stock       int       `json:"stock"`
	CreatedAt   time.Time `json:"created_at"`
	Quantity    int       `json:"quantity"`
}

type User struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Hash  []byte
}

type Discount struct {
	ID      int       `json:"discount_id"`
	Code    string    `json:"code"`
	Amount  int       `json:"amount"`
	Ends_at time.Time `json:"ends_at"`
}
