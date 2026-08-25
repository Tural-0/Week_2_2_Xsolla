package validation

import "errors"

var (
	ErrInvalidQuantity   = errors.New("quantity must be greater than 0")
	ErrInsufficientStock = errors.New("quantity exceeds available stock")
)

func QuantityIsPositive(quantity int) error {
	if quantity <= 0 {
		return ErrInvalidQuantity
	}

	return nil
}

func QuantityWithinStock(quantity, stock int) error {
	if quantity > stock {
		return ErrInsufficientStock
	}

	return nil
}
