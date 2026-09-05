package services

import (
	"context"
	"fmt"
	"time"

	"checkout-api/models"
	"checkout-api/validation"
)

type OrderStore interface {
	CreateOrder(
		ctx context.Context,
		userID int,
		items []models.LineItem,
		total int,
		status string,
		discount int,
	) (*models.Order, error)

	UpdateOrderStatus(
		ctx context.Context,
		orderID int,
		status string,
	) error

	GetDiscountDetails(
		ctx context.Context,
		discountCode string,
	) (models.Discount, error)
}

type PaymentResult struct {
	Success       bool   `json:"success"`
	TransactionID string `json:"transaction_id,omitempty"`
	Error         string `json:"error,omitempty"`
}

type OrderService struct {
	store OrderStore
}

func NewOrderService(store OrderStore) *OrderService {
	return &OrderService{store: store}
}

func (s *OrderService) PlaceOrder(
	ctx context.Context,
	userID int,
	items []models.LineItem,
	total int,
	discountCode string,
) (*models.Order, PaymentResult, error) {
	if err := validation.NonEmptyCart(items); err != nil {
		return nil, PaymentResult{}, err
	}

	discount, err := s.store.GetDiscountDetails(ctx, discountCode)
	if err != nil {
		return nil, PaymentResult{}, err
	}

	if err := validation.DiscountCheck(discount, time.Now()); err != nil {
		return nil, PaymentResult{}, err
	}

	calculatedTotal := 0

	for _, item := range items {
		calculatedTotal += item.Quantity * item.Price
	}

	if calculatedTotal != total {
		total = calculatedTotal
	}

	order, err := s.store.CreateOrder(
		ctx,
		userID,
		items,
		total,
		"pending",
		discount.Amount,
	)
	if err != nil {
		return nil, PaymentResult{}, err
	}

	paymentResult := MockProcessPayment(total)

	status := "paid"

	if !paymentResult.Success {
		status = "failed"
	}

	if paymentResult.Success && discount.Amount != 0 {
		status = fmt.Sprintf("paid (discount %d%%)", discount.Amount)
	}

	if err := s.store.UpdateOrderStatus(ctx, order.ID, status); err != nil {
		return nil, PaymentResult{}, err
	}

	order.Status = status

	return order, paymentResult, nil
}

// mockProcessPayment simulates a payment provider call.
func MockProcessPayment(amount int) PaymentResult {
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
