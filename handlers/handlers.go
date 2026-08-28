package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"checkout-api/models"
)

// ItemStore defines the data operations the handler needs.
type ItemStore interface {
	GetItems(ctx context.Context) ([]*models.Item, error)
	GetItem(ctx context.Context, ID int) (*models.Item, error)
	GetItemQuantityByID(ctx context.Context, userID int, itemID int) (int, error)
	GetItemsOffset(ctx context.Context, limit int, offset int) ([]models.Item, error)
	GetItemsCursor(ctx context.Context, limit int, cursor *int) ([]models.Item, error)

	CreateOrder(ctx context.Context, userID int, items []models.LineItem, total int, status string, discount int) (*models.Order, error)
	UpdateOrderStatus(ctx context.Context, orderID int, status string) error
	GetUserOrders(ctx context.Context, userID int) ([]models.Order, error)
	GetUserOrdersOffset(ctx context.Context, userID int, limit int, offset int) ([]models.Order, error)
	GetUserOrdersCursor(ctx context.Context, userID int, limit int, cursor *int) ([]models.Order, error)

	GetDiscountDetails(ctx context.Context, discountCode string) (models.Discount, error)

	CreateUserCart(ctx context.Context, cart *models.Cart) error
	UpsertCartItem(ctx context.Context, userID int, itemID int, quantity int) error
	GetUserCart(ctx context.Context, userID int) ([]models.CartItem, error)
	DeleteUserCart(ctx context.Context, userID int) error
	RemoveCartItem(ctx context.Context, userID int, itemID int) error

	SaveUser(ctx context.Context, email string, hash []byte) error
	FindUserByEmail(ctx context.Context, email string) (models.User, error)

	GetRefreshToken(ctx context.Context, token string) (int, bool, error)
	SaveRefreshToken(ctx context.Context, userID int, token string) error
	DeactivateRefreshToken(ctx context.Context, token string) error
}

var SigningSecret string = os.Getenv("JWT_SECRET")

// Handler holds dependencies for HTTP handlers.
type Handler struct {
	store            ItemStore
	idempotencyCache map[string]*IdempotencyRecord
}

// NewHandler creates a Handler with the given store.
func NewHandler(s ItemStore) *Handler {
	return &Handler{
		store:            s,
		idempotencyCache: make(map[string]*IdempotencyRecord),
	}
}

type IdempotencyRecord struct {
	Response   []byte
	StatusCode int
	Expiry     time.Time
}

// mockProcessPayment simulates a payment provider call.
func mockProcessPayment(amount int) PaymentResult {
	if amount > 0 && amount < 1000000 {
		return PaymentResult{
			Success:       true,
			TransactionID: fmt.Sprintf("txn_%d", time.Now().UnixNano()),
		}
	}
	return PaymentResult{
		Success: false,
		Error:   "Payment declined",
	}
}

// writeJSON encodes v as JSON and writes it to the response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
