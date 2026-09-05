package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"checkout-api/models"
	"checkout-api/validation"
)

type fakeOrderStore struct {
	discount        models.Discount
	discountErr     error
	createOrderErr  error
	updateStatusErr error

	createCalled bool
	updateCalled bool

	createdUserID   int
	createdItems    []models.LineItem
	createdTotal    int
	createdStatus   string
	createdDiscount int

	updatedOrderID int
	updatedStatus  string
}

func (f *fakeOrderStore) GetDiscountDetails(
	ctx context.Context,
	code string,
) (models.Discount, error) {
	return f.discount, f.discountErr
}

func (f *fakeOrderStore) CreateOrder(
	ctx context.Context,
	userID int,
	items []models.LineItem,
	total int,
	status string,
	discount int,
) (*models.Order, error) {
	f.createCalled = true
	f.createdUserID = userID
	f.createdItems = items
	f.createdTotal = total
	f.createdStatus = status
	f.createdDiscount = discount

	if f.createOrderErr != nil {
		return nil, f.createOrderErr
	}

	return &models.Order{
		ID:     1,
		UserID: userID,
		Items:  items,
		Total:  total,
		Status: status,
	}, nil
}

func (f *fakeOrderStore) UpdateOrderStatus(
	ctx context.Context,
	orderID int,
	status string,
) error {
	f.updateCalled = true
	f.updatedOrderID = orderID
	f.updatedStatus = status

	return f.updateStatusErr
}

func TestOrderService_PlaceOrder(t *testing.T) {
	tests := []struct {
		name        string
		items       []models.LineItem
		total       int
		discount    string
		store       *fakeOrderStore
		wantErr     error
		wantStatus  string
		wantPayment bool
	}{
		{
			name:  "empty cart",
			items: []models.LineItem{},
			store: &fakeOrderStore{
				discount: models.Discount{},
			},
			wantErr: validation.ErrNoItemInCart,
		},
		{
			name: "one item",
			items: []models.LineItem{
				{ItemID: 1, Quantity: 1, Price: 100},
			},
			total: 100,
			store: &fakeOrderStore{
				discount: models.Discount{},
			},
			wantStatus:  "paid",
			wantPayment: true,
		},
		{
			name: "many items",
			items: []models.LineItem{
				{ItemID: 1, Quantity: 2, Price: 100},
				{ItemID: 2, Quantity: 3, Price: 50},
			},
			total: 350,
			store: &fakeOrderStore{
				discount: models.Discount{},
			},
			wantStatus:  "paid",
			wantPayment: true,
		},
		{
			name: "payment boundary",
			items: []models.LineItem{
				{ItemID: 1, Quantity: 1, Price: 1000000},
			},
			total: 1000000,
			store: &fakeOrderStore{
				discount: models.Discount{},
			},
			wantStatus:  "failed",
			wantPayment: false,
		},
		{
			name: "expired discount",
			items: []models.LineItem{
				{ItemID: 1, Quantity: 1, Price: 100},
			},
			total:    100,
			discount: "OLD",
			store: &fakeOrderStore{
				discount: models.Discount{
					Code:    "OLD",
					Ends_at: time.Now().Add(-time.Hour),
				},
			},
			wantErr: validation.ErrLateDiscount,
		},
		{
			name: "store create error",
			items: []models.LineItem{
				{ItemID: 1, Quantity: 1, Price: 100},
			},
			total: 100,
			store: &fakeOrderStore{
				discount:       models.Discount{},
				createOrderErr: errors.New("create order failed"),
			},
			wantErr: errors.New("create order failed"),
		},
		{
			name: "total is recalculated",
			items: []models.LineItem{
				{ItemID: 1, Quantity: 2, Price: 100},
			},
			total: 999,
			store: &fakeOrderStore{
				discount: models.Discount{},
			},
			wantStatus:  "paid",
			wantPayment: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewOrderService(tt.store)

			order, payment, err := service.PlaceOrder(
				context.Background(),
				42,
				tt.items,
				tt.total,
				tt.discount,
			)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				if !errors.Is(err, tt.wantErr) &&
					err.Error() != tt.wantErr.Error() {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if order == nil {
				t.Fatal("expected order, got nil")
			}

			if order.Status != tt.wantStatus {
				t.Fatalf("expected status %q, got %q", tt.wantStatus, order.Status)
			}

			if payment.Success != tt.wantPayment {
				t.Fatalf(
					"expected payment success %v, got %v",
					tt.wantPayment,
					payment.Success,
				)
			}

			if !tt.store.createCalled {
				t.Fatal("expected CreateOrder to be called")
			}

			if !tt.store.updateCalled {
				t.Fatal("expected UpdateOrderStatus to be called")
			}
		})
	}
}
