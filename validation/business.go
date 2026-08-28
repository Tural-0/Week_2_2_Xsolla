package validation

import (
	"checkout-api/models"
	"errors"
	"net/mail"
	"reflect"
	"time"
)

var (
	ErrInvalidQuantity   = errors.New("quantity must be greater than 0")
	ErrInsufficientStock = errors.New("quantity exceeds available stock")
	ErrNoItemInCart      = errors.New("cannot place order with an empty cart")
	ErrNotAnArray        = errors.New("provided argument is not an array")
	ErrInvalidEmail      = errors.New("this email is invalid")
	ErrInvalidPassword   = errors.New("password does not fulfill the requirements")
	ErrInvalidDiscount   = errors.New("this discount code is not valid")
	ErrLateDiscount      = errors.New("this discount code has expired")
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

func DiscountCheck(discount models.Discount) error {
	if discount.Code == "" {
		return nil
	}

	if discount.Ends_at.Compare(time.Now()) == -1 {
		return ErrLateDiscount
	}
	return nil
}
