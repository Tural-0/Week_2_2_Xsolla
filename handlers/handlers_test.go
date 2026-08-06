package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"checkout-api/store"

	"github.com/golang-jwt/jwt/v5"
)

// testJWT signs a JWT for the given user ID using the handler's signing secret,
// for use in tests that exercise Authorization-protected endpoints.
func testJWT(t *testing.T, userID string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	})
	signed, err := token.SignedString([]byte(SigningSecret))
	if err != nil {
		t.Fatalf("failed to sign test jwt: %v", err)
	}
	return signed
}

func TestGetItems(t *testing.T) {
	s := store.NewInMemStore()
	h := NewHandler(s)

	tests := []struct {
		name       string
		method     string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "GET returns items",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
			wantBody:   "Laptop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/items", nil)
			w := httptest.NewRecorder()

			h.GetItems(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantBody != "" && !strings.Contains(w.Body.String(), tt.wantBody) {
				t.Errorf("body missing %q, got: %s", tt.wantBody, w.Body.String())
			}
		})
	}
}

func TestGetItemByID(t *testing.T) {
	s := store.NewInMemStore()
	h := NewHandler(s)

	tests := []struct {
		name       string
		itemIDPath string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "valid item",
			itemIDPath: "1",
			wantStatus: http.StatusOK,
			wantBody:   "Laptop",
		},
		{
			name:       "item not found",
			itemIDPath: "999",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid ID",
			itemIDPath: "abc",
			wantStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/items/"+tt.itemIDPath, nil)
			req.SetPathValue("item_id", tt.itemIDPath)
			w := httptest.NewRecorder()

			h.GetItemByID(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantBody != "" && !strings.Contains(w.Body.String(), tt.wantBody) {
				t.Errorf("body missing %q, got: %s", tt.wantBody, w.Body.String())
			}
		})
	}
}

func TestGetUserCart(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		setupCart  bool
		wantStatus int
		wantBody   string
	}{
		{
			name:       "cart does not exist - returns empty cart",
			userID:     "1",
			setupCart:  false,
			wantStatus: http.StatusOK,
			wantBody:   `"user_id":1,"items":[]`,
		},
		{
			name:       "cart exists - returns cart with items",
			userID:     "1",
			setupCart:  true,
			wantStatus: http.StatusOK,
			wantBody:   `"items":[{"id":1,"name":"Laptop"`,
		},
		{
			name:       "missing user ID header",
			userID:     "",
			setupCart:  false,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid user ID header",
			userID:     "invalid",
			setupCart:  false,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := store.NewInMemStore()
			h := NewHandler(s)

			if tt.setupCart {
				createReq := httptest.NewRequest(http.MethodPut, "/user/cart/items/1", strings.NewReader(`{"quantity":2}`))
				createReq.Header.Set("Authorization", "Bearer "+testJWT(t, "1"))
				createReq.Header.Set("Content-Type", "application/json")
				createReq.SetPathValue("item_id", "1")
				createW := httptest.NewRecorder()
				h.UpsertCartItem(createW, createReq)
			}

			req := httptest.NewRequest(http.MethodGet, "/user/cart", nil)
			if tt.userID != "" {
				req.Header.Set("X-User-ID", tt.userID)
			}
			w := httptest.NewRecorder()

			h.GetUserCart(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantBody != "" && !strings.Contains(w.Body.String(), tt.wantBody) {
				t.Errorf("body missing %q, got: %s", tt.wantBody, w.Body.String())
			}
		})
	}
}

func TestCreateOrder(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		idempotKey string
		body       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "valid order",
			userID:     "1",
			idempotKey: "key-001",
			body:       `{"line_items":[{"item_id":1,"quantity":1,"price":120000}],"total":120000}`,
			wantStatus: http.StatusCreated,
			wantBody:   `"status":"paid"`,
		},
		{
			name:       "empty items",
			userID:     "1",
			idempotKey: "key-002",
			body:       `{"line_items":[],"total":0}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing X-User-ID header",
			userID:     "",
			idempotKey: "key-003",
			body:       `{"line_items":[{"item_id":1,"quantity":1,"price":120000}],"total":120000}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing idempotency key",
			userID:     "1",
			idempotKey: "",
			body:       `{"line_items":[{"item_id":1,"quantity":1,"price":120000}],"total":120000}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid body",
			userID:     "1",
			idempotKey: "key-004",
			body:       `not-json`,

			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := store.NewInMemStore()
			h := NewHandler(s)

			req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.userID != "" {
				req.Header.Set("X-User-ID", tt.userID)
			}
			if tt.idempotKey != "" {
				req.Header.Set("Idempotency-Key", tt.idempotKey)
			}
			w := httptest.NewRecorder()

			h.CreateOrder(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantBody != "" && !strings.Contains(w.Body.String(), tt.wantBody) {
				t.Errorf("body missing %q, got: %s", tt.wantBody, w.Body.String())
			}
		})
	}
}
