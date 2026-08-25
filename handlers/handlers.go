package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"checkout-api/models"
)

// ItemStore defines the data operations the handler needs.
type ItemStore interface {
	GetItems(ctx context.Context) ([]*models.Item, error)
	GetItem(ctx context.Context, ID int) (*models.Item, error)
	GetItemQuantityByID(ctx context.Context, userID int, itemID int) (int, error)

	CreateOrder(ctx context.Context, userID int, items []models.LineItem, total int, status string) (*models.Order, error)
	UpdateOrderStatus(ctx context.Context, orderID int, status string) error
	GetUserOrders(ctx context.Context, userID int) ([]models.Order, error)

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

func (h *Handler) CreateUserCart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var cartReq CreateCartRequest

	err := json.NewDecoder(r.Body).Decode(&cartReq)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	///////////////////
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		http.Error(w, "missing X-User-ID header", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "invalid X-User-ID header", http.StatusBadRequest)
		return
	}
	///////////////////

	//userID := r.Context().Value("userID").(int) // if JWT is a middleware

	items, err := h.store.GetUserCart(r.Context(), userID)
	if len(items) != 0 {
		http.Error(w, "cart already exists", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if cartReq.Quantity <= 0 {
		http.Error(w, "quantity must be greater than 0", http.StatusBadRequest)
		return
	}

	item, err := h.store.GetItem(r.Context(), cartReq.ItemID)
	if err != nil || item == nil {
		http.Error(w, "item with that ID doesn't exist", http.StatusNotFound)
		return
	}

	if cartReq.Quantity > item.Stock {
		http.Error(w, "quantity is greater than stock", http.StatusBadRequest)
		return
	}

	cart := models.Cart{
		UserID:   userID,
		ItemID:   cartReq.ItemID,
		Quantity: cartReq.Quantity,
	}
	err = h.store.CreateUserCart(r.Context(), &cart)
	if err != nil {
		http.Error(w, "Failed to create cart", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, cart)
}

func (h *Handler) UpsertCartItem(w http.ResponseWriter, r *http.Request) {

	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		http.Error(w, "missing X-User-ID header", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "invalid X-User-ID header", http.StatusBadRequest)
		return
	}

	//userID := r.Context().Value("userID").(int) // if JWT is a middleware

	itemIDStr := r.PathValue("item_id")
	if itemIDStr == "" {
		http.Error(w, "missing item_id", http.StatusBadRequest)
		return
	}

	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, ErrorMessageResponse{
			Message: "item_id must be integer",
		})
		return
	}

	var req UpsertCartItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Quantity <= 0 {
		http.Error(w, "quantity must be greater than 0", http.StatusBadRequest)
		return
	}

	if err := h.store.UpsertCartItem(r.Context(), userID, itemID, req.Quantity); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}

func (h *Handler) RemoveCartItem(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		http.Error(w, "missing X-User-ID header", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "invalid X-User-ID header", http.StatusBadRequest)
		return
	}

	//userID := r.Context().Value("userID").(int) // if JWT is a middleware

	itemIDStr := r.PathValue("item_id")
	if itemIDStr == "" {
		http.Error(w, "missing item_id", http.StatusBadRequest)
		return
	}

	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, ErrorMessageResponse{
			Message: "item_id must be integer",
		})
		return
	}

	if err := h.store.RemoveCartItem(r.Context(), userID, itemID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetUserCart(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		http.Error(w, "missing X-User-ID header", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "invalid X-User-ID header", http.StatusBadRequest)
		return
	}

	//userID := r.Context().Value("userID").(int) // if JWT is a middleware

	cart, err := h.store.GetUserCart(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	items := make([]CartItemResponse, 0, len(cart))
	for _, item := range cart {
		items = append(items, CartItemResponse{
			ID:          item.ID,
			Name:        item.Name,
			Description: item.Description,
			Price:       item.Price,
			Stock:       item.Stock,
			CreatedAt:   item.CreatedAt,
			Quantity:    item.Quantity,
		})
	}

	writeJSON(w, http.StatusOK, CartResponse{
		UserID: userID,
		Items:  items,
	})
}

func (h *Handler) DeleteUserCart(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		http.Error(w, "missing X-User-ID header", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "invalid X-User-ID header", http.StatusBadRequest)
		return
	}

	//userID := r.Context().Value("userID").(int) // if JWT is a middleware

	err = h.store.DeleteUserCart(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		http.Error(w, "missing X-User-ID header", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "invalid X-User-ID header", http.StatusBadRequest)
		return
	}

	//userID := r.Context().Value("userID").(int) // if JWT is a middleware

	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		http.Error(w, "Idempotency-Key header is required", http.StatusBadRequest)
		return
	}

	if record, exists := h.idempotencyCache[idempotencyKey]; exists {
		if time.Now().Before(record.Expiry) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(record.StatusCode)
			w.Write(record.Response)
			return
		}
		delete(h.idempotencyCache, idempotencyKey)
	}

	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.LineItems) == 0 {
		http.Error(w, "items must not be empty", http.StatusBadRequest)
		return
	}

	items := make([]models.LineItem, 0, len(req.LineItems))
	for _, i := range req.LineItems {
		items = append(items, models.LineItem{
			ItemID:   i.ItemID,
			Quantity: i.Quantity,
			Price:    i.Price,
		})
	}

	order, err := h.store.CreateOrder(r.Context(), userID, items, req.Total, "pending")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	paymentResult := mockProcessPayment(req.Total)

	status := "paid"
	if !paymentResult.Success {
		status = "failed"
	}

	if err := h.store.UpdateOrderStatus(r.Context(), order.ID, status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	order.Status = status

	responseData := map[string]any{
		"order":   order,
		"payment": paymentResult,
	}

	statusCode := http.StatusCreated
	if !paymentResult.Success {
		statusCode = http.StatusPaymentRequired
	}

	responseBody, _ := json.Marshal(responseData)
	h.idempotencyCache[idempotencyKey] = &IdempotencyRecord{
		Response:   responseBody,
		StatusCode: statusCode,
		Expiry:     time.Now().Add(24 * time.Hour),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(responseBody)
}

// GetItems handles GET /items — returns all available items.
func (h *Handler) GetItems(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.GetItems(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, items)
}

// GetItemByID handles GET /items/{id} — returns a single i`tem.
func (h *Handler) GetItemByID(w http.ResponseWriter, r *http.Request) {
	itemIDStr := r.PathValue("item_id")
	if itemIDStr == "" {
		http.Error(w, "URL doesn't contain item ID", http.StatusUnprocessableEntity)
		return
	}

	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, ErrorMessageResponse{
			Message: "item_id must be an integer",
		})
		return
	}

	item, err := h.store.GetItem(r.Context(), itemID)
	if err != nil {
		fmt.Printf("error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if item == nil {
		writeJSON(w, http.StatusNotFound, nil)
		return
	}

	writeJSON(w, http.StatusOK, item)
}

// writeJSON encodes v as JSON and writes it to the response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) LoginUser(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	user, err := h.store.FindUserByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, ErrorMessageResponse{
				Message: "user does not exist",
			})
			return
		}
		fmt.Printf("cannot query %q", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = bcrypt.CompareHashAndPassword(user.Hash, []byte(req.Password))
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	// issue jwt
	//fifteenAfter := time.Now().Add(15 * time.Minute)
	//token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
	//	ExpiresAt: jwt.NewNumericDate(fifteenAfter),
	//	Subject:   strconv.Itoa(user.ID),
	//	IssuedAt:  jwt.NewNumericDate(time.Now()),
	//})

	//signedString, err := token.SignedString([]byte(SigningSecret))
	signedString, err := GenerateJWT(user.ID)
	if err != nil {
		fmt.Printf("cannot generate signed string %q", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// store session(refresh token)
	// TODO: do it yourself
	// generate a random string(bonus: if you use a CSPRNG to generate a random sequence of bytes)
	// insert into refresh_tokens (token_value, is_active) values ("sOmERANdomlYGeNERATEDstRing", 1)
	refreshToken, err := GenerateRefreshToken()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = h.store.SaveRefreshToken(r.Context(), user.ID, refreshToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, AuthResponse{
		JWT:          signedString,
		RefreshToken: refreshToken,
	})
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	// validate email
	_, err = mail.ParseAddress(req.Email)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, ErrorMessageResponse{
			Message: "invalid email",
		})
		return
	}

	pwlen := len(req.Password)
	// validate password
	if pwlen < 12 || pwlen > 25 {
		writeJSON(w, http.StatusUnprocessableEntity, ErrorMessageResponse{
			Message: "password is too short or too long",
		})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = h.store.SaveUser(r.Context(), req.Email, hash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, nil)
}

/////////////////////  JWT TOKENS  ///////////////////////////

func GenerateJWT(userID int) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}

func GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 32) // 32 byte for 256 bits, means 2^256 possible

	_, err := rand.Read(bytes) // CSPRNG
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil // convert to hex
}

func (h *Handler) IssueJWT(w http.ResponseWriter, r *http.Request) {
	// TODO: implement issueing of new JWT with refresh token
	// check if refresh_token exists in the db and still active
	// generate a new JWT
	// generate a new random string (bonus: if you use a CSPRNG to generate a random sequence of bytes) as refresh_token
	// save new refresh token in db
	// deactivate old refresh token

	var req RefreshRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	userID, active, err := h.store.GetRefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		http.Error(w, "refresh token not found", http.StatusUnauthorized)
		return
	}

	if !active {
		http.Error(w, "refresh token inactive", http.StatusUnauthorized)
		return
	}

	jwtToken, err := GenerateJWT(userID)
	if err != nil {
		http.Error(w, "failed to generate jwt", http.StatusInternalServerError)
		return
	}

	newRefreshToken, err := GenerateRefreshToken()
	if err != nil {
		http.Error(w, "failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	err = h.store.SaveRefreshToken(r.Context(), userID, newRefreshToken)
	if err != nil {
		http.Error(w, "failed to save refresh token", http.StatusInternalServerError)
		return
	}

	err = h.store.DeactivateRefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		http.Error(w, "failed to deactivate refresh token", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"jwt":           jwtToken,
		"refresh_token": newRefreshToken,
	})

}

////////////////////////////////////////////////////////////////

func (h *Handler) GetItemQuantityByID(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		http.Error(w, "missing X-User-ID header", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "invalid X-User-ID header", http.StatusBadRequest)
		return
	}

	itemIDStr := r.PathValue("item_id")
	if itemIDStr == "" {
		http.Error(w, "Not Found", http.StatusUnprocessableEntity)
		return
	}

	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, ErrorMessageResponse{
			Message: "item_id must be an integer",
		})
		return
	}

	quantity, err := h.store.GetItemQuantityByID(r.Context(), userID, itemID)
	if quantity <= 0 {
		writeJSON(w, http.StatusOK, quantity)
		return
	}

	if err != nil {
		fmt.Printf("error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, quantity)
}

/////////////////////////////////////////////////////////////////////////////////

func (h *Handler) GetUserOrders(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		http.Error(w, "missing X-User-ID header", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "invalid X-User-ID header", http.StatusBadRequest)
		return
	}

	//userID := r.Context().Value("userID").(int) // if JWT is a middleware

	orders, err := h.store.GetUserOrders(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ordersResponse := make([]OrderResponse, 0, len(orders))
	for _, order := range orders {
		items := make([]LinteItemResponse, 0, len(order.Items))

		for _, item := range order.Items {
			items = append(items, LinteItemResponse{
				ItemID:   item.ItemID,
				Quantity: item.Quantity,
				Price:    item.Price,
			})
		}

		ordersResponse = append(ordersResponse, OrderResponse{
			ID:     order.ID,
			UserID: order.UserID,
			Items:  items,
			Total:  order.Total,
			Status: order.Status,
		})
	}

	writeJSON(w, http.StatusOK, orders)
}
