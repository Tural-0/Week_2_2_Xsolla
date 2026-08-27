package validation

import (
	"errors"
	"net/mail"
	"reflect"
)

var (
	ErrInvalidQuantity   = errors.New("quantity must be greater than 0")
	ErrInsufficientStock = errors.New("quantity exceeds available stock")
	ErrNoItemInCart      = errors.New("cannot place order with an empty cart")
	ErrNotAnArray        = errors.New("provided argument is not an array")
	ErrInvalidEmail      = errors.New("this email is invalid")
	ErrInvalidPassword   = errors.New("password does not fulfill the requirements")
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

func NonEmptyCart(items any) error {

	val := reflect.ValueOf(items)

	if val.Kind() != reflect.Array && val.Kind() != reflect.Slice {
		return ErrNotAnArray
	}

	if val.Len() == 0 {
		return ErrNoItemInCart
	}

	return nil
}

func EmailCheck(email string) error {
	_, err := mail.ParseAddress(email)
	if err != nil {
		return ErrInvalidEmail
	}

	return nil
}

func PasswordCheck(pass string) error {
	pwlen := len(pass)
	if pwlen < 12 || pwlen > 25 {
		return ErrInvalidPassword
	}

	return nil
}
