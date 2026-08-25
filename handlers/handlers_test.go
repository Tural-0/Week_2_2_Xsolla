package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"checkout-api/models"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// testStore is a small fake store used only for handler tests.
// It avoids depending on PostgreSQL or the incomplete in-memory store.
type testStore struct {
	items         map[int]*models.Item
	orders        map[int]*models.Order
	carts         map[int][]models.CartItem
	users         map[string]models.User
	refreshTokens map[string]struct {
		userID int
		active bool
	}

	nextOrderID int
	nextUserID  int
}

func newTestStore() *testStore {
	return &testStore{
		items: map[int]*models.Item{
			1: {
				ID:          1,
				Name:        "Laptop",
				Description: "A fast laptop",
				Price:       120000,
				Stock:       10,
			},
			2: {
				ID:          2,
				Name:        "Mouse",
				Description: "Wireless mouse",
				Price:       2500,
				Stock:       50,
			},
		},
		orders: make(map[int]*models.Order),
		carts:  make(map[int][]models.CartItem),
		users:  make(map[string]models.User),
		refreshTokens: make(map[string]struct {
			userID int
			active bool
		}),
		nextOrderID: 1,
		nextUserID:  1,
	}
}

// --------------------------------------------------
// Store methods
// --------------------------------------------------

func (s *testStore) GetItems(_ context.Context) ([]*models.Item, error) {
	items := make([]*models.Item, 0, len(s.items))

	for _, item := range s.items {
		items = append(items, item)
	}

	return items, nil
}

func (s *testStore) GetItem(_ context.Context, id int) (*models.Item, error) {
	return s.items[id], nil
}

func (s *testStore) GetItemQuantityByID(_ context.Context, userID int, itemID int) (int, error) {
	for _, item := range s.carts[userID] {
		if item.ID == itemID {
			return item.Quantity, nil
		}
	}

	return 0, nil
}

func (s *testStore) CreateOrder(
	_ context.Context,
	userID int,
	items []models.LineItem,
	total int,
	status string,
) (*models.Order, error) {
	order := &models.Order{
		ID:     s.nextOrderID,
		UserID: userID,
		Items:  items,
		Total:  total,
		Status: status,
	}

	s.orders[order.ID] = order
	s.nextOrderID++

	return order, nil
}

func (s *testStore) UpdateOrderStatus(
	_ context.Context,
	orderID int,
	status string,
) error {
	order, ok := s.orders[orderID]
	if !ok {
		return nil
	}

	order.Status = status
	return nil
}

func (s *testStore) GetUserOrders(
	_ context.Context,
	userID int,
) ([]models.Order, error) {
	orders := make([]models.Order, 0)

	for _, order := range s.orders {
		if order.UserID == userID {
			orders = append(orders, *order)
		}
	}

	return orders, nil
}

func (s *testStore) CreateUserCart(
	_ context.Context,
	cart *models.Cart,
) error {
	s.carts[cart.UserID] = []models.CartItem{
		{
			ID:       cart.ItemID,
			Name:     s.items[cart.ItemID].Name,
			Price:    s.items[cart.ItemID].Price,
			Stock:    s.items[cart.ItemID].Stock,
			Quantity: cart.Quantity,
		},
	}

	return nil
}

func (s *testStore) UpsertCartItem(
	_ context.Context,
	userID int,
	itemID int,
	quantity int,
) error {
	item, ok := s.items[itemID]
	if !ok {
		return nil
	}

	for i := range s.carts[userID] {
		if s.carts[userID][i].ID == itemID {
			s.carts[userID][i].Quantity = quantity
			return nil
		}
	}

	s.carts[userID] = append(s.carts[userID], models.CartItem{
		ID:       item.ID,
		Name:     item.Name,
		Price:    item.Price,
		Stock:    item.Stock,
		Quantity: quantity,
	})

	return nil
}

func (s *testStore) GetUserCart(
	_ context.Context,
	userID int,
) ([]models.CartItem, error) {
	return s.carts[userID], nil
}

func (s *testStore) DeleteUserCart(
	_ context.Context,
	userID int,
) error {
	delete(s.carts, userID)
	return nil
}

func (s *testStore) RemoveCartItem(
	_ context.Context,
	userID int,
	itemID int,
) error {
	items := s.carts[userID]

	for i, item := range items {
		if item.ID == itemID {
			s.carts[userID] = append(items[:i], items[i+1:]...)
			return nil
		}
	}

	return nil
}

func (s *testStore) SaveUser(
	_ context.Context,
	email string,
	hash []byte,
) error {
	if _, exists := s.users[email]; exists {
		return nil
	}

	s.users[email] = models.User{
		ID:    s.nextUserID,
		Email: email,
		Hash:  hash,
	}

	s.nextUserID++

	return nil
}

func (s *testStore) FindUserByEmail(
	_ context.Context,
	email string,
) (models.User, error) {
	user, ok := s.users[email]

	if !ok {
		return models.User{}, pgx.ErrNoRows
	}

	return user, nil
}

func (s *testStore) GetRefreshToken(
	_ context.Context,
	token string,
) (int, bool, error) {
	refreshToken, ok := s.refreshTokens[token]

	if !ok {
		return 0, false, pgx.ErrNoRows
	}

	return refreshToken.userID, refreshToken.active, nil
}

func (s *testStore) SaveRefreshToken(
	_ context.Context,
	userID int,
	token string,
) error {
	s.refreshTokens[token] = struct {
		userID int
		active bool
	}{
		userID: userID,
		active: true,
	}

	return nil
}

func (s *testStore) DeactivateRefreshToken(
	_ context.Context,
	token string,
) error {
	refreshToken, ok := s.refreshTokens[token]

	if ok {
		refreshToken.active = false
		s.refreshTokens[token] = refreshToken
	}

	return nil
}

// --------------------------------------------------
// Tests
// --------------------------------------------------

func TestGetItems(t *testing.T) {
	store := newTestStore()
	handler := NewHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/items", nil)
	rec := httptest.NewRecorder()

	handler.GetItems(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "Laptop") {
		t.Errorf("expected response to contain Laptop")
	}
}

func TestGetItemByID(t *testing.T) {
	tests := []struct {
		name       string
		itemID     string
		wantStatus int
	}{
		{
			name:       "existing item",
			itemID:     "1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "item not found",
			itemID:     "999",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid item id",
			itemID:     "abc",
			wantStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestStore()
			handler := NewHandler(store)

			req := httptest.NewRequest(
				http.MethodGet,
				"/items/"+tt.itemID,
				nil,
			)

			req.SetPathValue("item_id", tt.itemID)

			rec := httptest.NewRecorder()

			handler.GetItemByID(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf(
					"expected status %d, got %d",
					tt.wantStatus,
					rec.Code,
				)
			}
		})
	}
}

func TestCreateUserCart(t *testing.T) {
	store := newTestStore()
	handler := NewHandler(store)

	req := httptest.NewRequest(
		http.MethodPost,
		"/user/carts",
		strings.NewReader(`{"item_id":1,"quantity":2}`),
	)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "1")

	rec := httptest.NewRecorder()

	handler.CreateUserCart(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
}

func TestGetUserCart(t *testing.T) {
	store := newTestStore()
	handler := NewHandler(store)

	// Create cart first.
	store.carts[1] = []models.CartItem{
		{
			ID:       1,
			Name:     "Laptop",
			Price:    120000,
			Stock:    10,
			Quantity: 2,
		},
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/user/cart",
		nil,
	)

	req.Header.Set("X-User-ID", "1")

	rec := httptest.NewRecorder()

	handler.GetUserCart(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "Laptop") {
		t.Errorf("expected cart to contain Laptop")
	}
}

func TestGetUserCartWithoutUserID(t *testing.T) {
	store := newTestStore()
	handler := NewHandler(store)

	req := httptest.NewRequest(
		http.MethodGet,
		"/user/cart",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.GetUserCart(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestUpsertCartItem(t *testing.T) {
	store := newTestStore()
	handler := NewHandler(store)

	req := httptest.NewRequest(
		http.MethodPatch,
		"/user/cart/items/1",
		strings.NewReader(`{"quantity":3}`),
	)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "1")
	req.SetPathValue("item_id", "1")

	rec := httptest.NewRecorder()

	handler.UpsertCartItem(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNoContent,
			rec.Code,
		)
	}

	if len(store.carts[1]) != 1 {
		t.Fatalf("expected one cart item")
	}

	if store.carts[1][0].Quantity != 3 {
		t.Errorf(
			"expected quantity 3, got %d",
			store.carts[1][0].Quantity,
		)
	}
}

func TestRemoveCartItem(t *testing.T) {
	store := newTestStore()
	handler := NewHandler(store)

	store.carts[1] = []models.CartItem{
		{
			ID:       1,
			Name:     "Laptop",
			Quantity: 2,
		},
	}

	req := httptest.NewRequest(
		http.MethodDelete,
		"/user/cart/items/1",
		nil,
	)

	req.Header.Set("X-User-ID", "1")
	req.SetPathValue("item_id", "1")

	rec := httptest.NewRecorder()

	handler.RemoveCartItem(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNoContent,
			rec.Code,
		)
	}

	if len(store.carts[1]) != 0 {
		t.Errorf("expected cart item to be removed")
	}
}

func TestDeleteUserCart(t *testing.T) {
	store := newTestStore()
	handler := NewHandler(store)

	store.carts[1] = []models.CartItem{
		{
			ID:       1,
			Name:     "Laptop",
			Quantity: 2,
		},
	}

	req := httptest.NewRequest(
		http.MethodDelete,
		"/user/cart",
		nil,
	)

	req.Header.Set("X-User-ID", "1")

	rec := httptest.NewRecorder()

	handler.DeleteUserCart(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNoContent,
			rec.Code,
		)
	}

	if _, ok := store.carts[1]; ok {
		t.Errorf("expected cart to be deleted")
	}
}

func TestCreateOrder(t *testing.T) {
	store := newTestStore()
	handler := NewHandler(store)

	body := `{
		"line_items": [
			{
				"item_id": 1,
				"quantity": 1,
				"price": 120000
			}
		],
		"total": 120000
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/orders",
		strings.NewReader(body),
	)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "1")
	req.Header.Set("Idempotency-Key", "test-key-1")

	rec := httptest.NewRecorder()

	handler.CreateOrder(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusCreated,
			rec.Code,
			rec.Body.String(),
		)
	}

	if !strings.Contains(rec.Body.String(), `"status":"paid"`) {
		t.Errorf("expected order status to be paid")
	}
}

func TestCreateOrderIdempotency(t *testing.T) {
	store := newTestStore()
	handler := NewHandler(store)

	body := `{
		"line_items": [
			{
				"item_id": 1,
				"quantity": 1,
				"price": 120000
			}
		],
		"total": 120000
	}`

	// First request.
	req1 := httptest.NewRequest(
		http.MethodPost,
		"/orders",
		strings.NewReader(body),
	)

	req1.Header.Set("X-User-ID", "1")
	req1.Header.Set("Idempotency-Key", "same-key")

	rec1 := httptest.NewRecorder()

	handler.CreateOrder(rec1, req1)

	if rec1.Code != http.StatusCreated {
		t.Fatalf("first request failed: %d", rec1.Code)
	}

	// Second request with the same key.
	req2 := httptest.NewRequest(
		http.MethodPost,
		"/orders",
		strings.NewReader(body),
	)

	req2.Header.Set("X-User-ID", "1")
	req2.Header.Set("Idempotency-Key", "same-key")

	rec2 := httptest.NewRecorder()

	handler.CreateOrder(rec2, req2)

	if rec2.Code != http.StatusCreated {
		t.Fatalf("second request failed: %d", rec2.Code)
	}

	if len(store.orders) != 1 {
		t.Errorf(
			"expected exactly 1 order, got %d",
			len(store.orders),
		)
	}
}

func TestGetItemQuantity(t *testing.T) {
	store := newTestStore()
	handler := NewHandler(store)

	store.carts[1] = []models.CartItem{
		{
			ID:       1,
			Name:     "Laptop",
			Quantity: 3,
		},
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/itemQuantity/1",
		nil,
	)

	req.Header.Set("X-User-ID", "1")
	req.SetPathValue("item_id", "1")

	rec := httptest.NewRecorder()

	handler.GetItemQuantityByID(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			rec.Code,
		)
	}

	if !strings.Contains(rec.Body.String(), "3") {
		t.Errorf("expected quantity 3, got %s", rec.Body.String())
	}
}

func TestGetUserOrders(t *testing.T) {
	store := newTestStore()
	handler := NewHandler(store)

	store.orders[1] = &models.Order{
		ID:     1,
		UserID: 1,
		Total:  120000,
		Status: "paid",
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/user/orders",
		nil,
	)

	req.Header.Set("X-User-ID", "1")

	rec := httptest.NewRecorder()

	handler.GetUserOrders(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			rec.Code,
		)
	}
}

func TestCreateUser(t *testing.T) {
	store := newTestStore()
	handler := NewHandler(store)

	req := httptest.NewRequest(
		http.MethodPost,
		"/signup",
		strings.NewReader(`{
			"email":"test@example.com",
			"password":"password123456"
		}`),
	)

	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	handler.CreateUser(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusCreated,
			rec.Code,
		)
	}

	if _, ok := store.users["test@example.com"]; !ok {
		t.Errorf("expected user to be created")
	}
}

func TestLoginUser(t *testing.T) {
	store := newTestStore()
	handler := NewHandler(store)

	password := "password123456"

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)

	if err != nil {
		t.Fatalf("failed to create test password hash: %v", err)
	}

	store.users["test@example.com"] = models.User{
		ID:    1,
		Email: "test@example.com",
		Hash:  hash,
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/login",
		strings.NewReader(`{
			"email":"test@example.com",
			"password":"password123456"
		}`),
	)

	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	handler.LoginUser(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusOK,
			rec.Code,
			rec.Body.String(),
		)
	}

	if !strings.Contains(rec.Body.String(), "jwt") {
		t.Errorf("expected response to contain jwt")
	}

	if !strings.Contains(rec.Body.String(), "refresh_token") {
		t.Errorf("expected response to contain refresh_token")
	}
}

func TestIssueJWT(t *testing.T) {
	store := newTestStore()
	handler := NewHandler(store)

	// Existing active refresh token.
	store.refreshTokens["old-refresh-token"] = struct {
		userID int
		active bool
	}{
		userID: 1,
		active: true,
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/token",
		strings.NewReader(`{
			"refresh_token":"old-refresh-token"
		}`),
	)

	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	handler.IssueJWT(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusOK,
			rec.Code,
			rec.Body.String(),
		)
	}

	if !strings.Contains(rec.Body.String(), "jwt") {
		t.Errorf("expected response to contain jwt")
	}

	if !strings.Contains(rec.Body.String(), "refresh_token") {
		t.Errorf("expected response to contain refresh_token")
	}

	oldToken := store.refreshTokens["old-refresh-token"]

	if oldToken.active {
		t.Errorf("expected old refresh token to be inactive")
	}
}
