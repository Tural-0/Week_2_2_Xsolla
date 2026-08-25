package handlers

type UpsertCartItemRequest struct {
	Quantity *int `json:"quantity"`
}

type CreateOrderRequest struct {
	LineItems []LineItemRequest `json:"line_items"`
	Total     int               `json:"total"`
}

type LineItemRequest struct {
	ItemID   int `json:"item_id"`
	Quantity int `json:"quantity"`
	Price    int `json:"price"`
}

type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type CreateCartRequest struct {
	ItemID   int `json:"item_id"`
	Quantity int `json:"quantity"`
}
