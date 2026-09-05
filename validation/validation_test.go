package validation

import (
	"checkout-api/models"
	"errors"
	"testing"
	"time"
)

func TestDiscountCheck(t *testing.T) {
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		discount models.Discount
		wantErr  error
	}{
		{
			name: "no discount",
			discount: models.Discount{
				Code:    "",
				Ends_at: now.Add(-time.Hour),
			},
			wantErr: nil,
		},
		{
			name: "valid discount",
			discount: models.Discount{
				Code:    "SAVE10",
				Ends_at: now.Add(time.Hour),
			},
			wantErr: nil,
		},
		{
			name: "expired discount",
			discount: models.Discount{
				Code:    "SAVE10",
				Ends_at: now.Add(-time.Hour),
			},
			wantErr: ErrLateDiscount,
		},
		{
			name: "discount expires in one second",
			discount: models.Discount{
				Code:    "SAVE10",
				Ends_at: now.Add(time.Second),
			},
			wantErr: nil,
		},
		{
			name: "empty code ignores expired date",
			discount: models.Discount{
				Code:    "",
				Ends_at: now.Add(-24 * time.Hour),
			},
			wantErr: nil,
		},
		{
			name: "discount expires far in the future",
			discount: models.Discount{
				Code:    "SAVE50",
				Ends_at: now.Add(24 * time.Hour),
			},
			wantErr: nil,
		},
		{
			name: "discount expires exactly now",
			discount: models.Discount{
				Code:    "SAVE10",
				Ends_at: now,
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DiscountCheck(tt.discount, now)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf(
					"DiscountCheck() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}
		})
	}
}

func TestNonEmptyCart(t *testing.T) {
	tests := []struct {
		name    string
		items   any
		wantErr error
	}{
		{
			name:    "non-empty slice",
			items:   []int{1, 2, 3},
			wantErr: nil,
		},
		{
			name:    "non-empty array",
			items:   [3]int{1, 2, 3},
			wantErr: nil,
		},
		{
			name:    "empty slice",
			items:   []int{},
			wantErr: ErrNoItemInCart,
		},
		{
			name:    "empty array",
			items:   [0]int{},
			wantErr: ErrNoItemInCart,
		},
		{
			name:    "string instead of array or slice",
			items:   "cart",
			wantErr: ErrNotAnArray,
		},
		{
			name:    "integer instead of array or slice",
			items:   123,
			wantErr: ErrNotAnArray,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NonEmptyCart(tt.items)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf(
					"NonEmptyCart() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}
		})
	}
}

func TestQuantityIsPositive(t *testing.T) {
	tests := []struct {
		name     string
		quantity int
		wantErr  error
	}{
		{
			name:     "negative quantity",
			quantity: -1,
			wantErr:  ErrInvalidQuantity,
		},
		{
			name:     "zero quantity",
			quantity: 0,
			wantErr:  ErrInvalidQuantity,
		},
		{
			name:     "positive quantity",
			quantity: 1,
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := QuantityIsPositive(tt.quantity)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf(
					"QuantityIsPositive() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}
		})
	}
}

func TestQuantityWithinStock(t *testing.T) {
	tests := []struct {
		name     string
		quantity int
		stock    int
		wantErr  error
	}{
		{
			name:     "quantity below stock",
			quantity: 5,
			stock:    10,
			wantErr:  nil,
		},
		{
			name:     "quantity exactly equals stock",
			quantity: 10,
			stock:    10,
			wantErr:  nil,
		},
		{
			name:     "quantity exceeds stock",
			quantity: 11,
			stock:    10,
			wantErr:  ErrInsufficientStock,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := QuantityWithinStock(tt.quantity, tt.stock)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf(
					"QuantityWithinStock() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}
		})
	}
}

func TestEmailCheck(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr error
	}{
		{
			name:    "valid email",
			email:   "user@example.com",
			wantErr: nil,
		},
		{
			name:    "valid email with dot",
			email:   "john.doe@example.com",
			wantErr: nil,
		},
		{
			name:    "invalid email",
			email:   "invalid-email",
			wantErr: ErrInvalidEmail,
		},
		{
			name:    "missing local part",
			email:   "@example.com",
			wantErr: ErrInvalidEmail,
		},
		{
			name:    "empty email",
			email:   "",
			wantErr: ErrInvalidEmail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := EmailCheck(tt.email)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf(
					"EmailCheck() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}
		})
	}
}

func TestPasswordCheck(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{
			name:     "password too short",
			password: "12345678901",
			wantErr:  ErrInvalidPassword,
		},
		{
			name:     "password with exactly 12 characters",
			password: "123456789012",
			wantErr:  nil,
		},
		{
			name:     "password with exactly 25 characters",
			password: "1234567890123456789012345",
			wantErr:  nil,
		},
		{
			name:     "password too long",
			password: "12345678901234567890123456",
			wantErr:  ErrInvalidPassword,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := PasswordCheck(tt.password)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf(
					"PasswordCheck() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}
		})
	}
}
