package validation

import "errors"

var (
	ErrInvalidItemID = errors.New("item_id must be a positive integer")
)

func ItemID(id int) error {
	if id <= 0 {
		return ErrInvalidItemID
	}

	return nil
}

func QuantityProvided(quantity *int) error {
	if quantity == nil {
		return errors.New("quantity is required")
	}

	return nil
}
