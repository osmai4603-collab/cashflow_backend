package ecommerce

import (
	"errors"
	"fmt"
)

var (
	ErrCartNotFound      = errors.New("cart not found")
	ErrProductOutOfStock = errors.New("product out of stock for web sales")
	ErrInvalidCoupon     = errors.New("invalid or expired coupon code")
	ErrCheckoutFailed    = errors.New("checkout process failed")
)

func WrapErrorf(err error, format string, args ...interface{}) error {
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}
