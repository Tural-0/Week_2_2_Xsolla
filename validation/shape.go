package validation

import (
	"errors"
	"strings"
)

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

func RequiredString(value, field string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New(field + " is required")
	}

	return nil
}
