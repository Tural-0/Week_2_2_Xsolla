package handlers

import "time"

type ErrorMessageResponse struct {
	Message string `json:"message"`
}

// PaymentResult represents a response from the payment provider.
type PaymentResult struct {
	Success       bool   `json:"success"`
	TransactionID string `json:"transaction_id,omitempty"`
	Error         string `json:"error,omitempty"`
}

type ItemResponse struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       int       `json:"price"` // Price in cents
	Stock       int       `json:"stock"`
	CreatedAt   time.Time `json:"created_at"`
}

type LinteItemResponse struct {
	ItemID   int `json:"item_id"`
	Quantity int `json:"quantity"`
	Price    int `json:"price"` // Price at time of adding
}

type OrderResponse struct {
	ID     int                 `json:"id"`
	UserID int                 `json:"user_id"`
	Items  []LinteItemResponse `json:"line_items"`
	Total  int                 `json:"total"`
	Status string              `json:"status"` // pending, paid, failed
}

type CartResponse struct {
	UserID int                `json:"user_id"`
	Items  []CartItemResponse `json:"items"`
}

type CartItemResponse struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       int       `json:"price"` // Price in cents
	Stock       int       `json:"stock"`
	CreatedAt   time.Time `json:"created_at"`
	Quantity    int       `json:"quantity"`
}

type AuthResponse struct {
	JWT          string `json:"jwt"`
	RefreshToken string `json:"refresh_token"`
}
