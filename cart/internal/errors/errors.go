package errors

import (
	"errors"
	"fmt"
)

var (
	ErrProductNotFound   = errors.New("product not found")
	ErrInsufficientStock = errors.New("insufficient stock")
)

type ProductNotFoundError struct {
	SKU uint32
}

func (e *ProductNotFoundError) Error() string {
	return fmt.Sprintf("product with sku=%d not found", e.SKU)
}

func NewProductNotFoundError(sku uint32) error {
	return &ProductNotFoundError{SKU: sku}
}

func (e *ProductNotFoundError) Unwrap() error {
	return ErrProductNotFound
}

type ItemNotFoundError struct {
	SKU uint32
}

func (e *ItemNotFoundError) Error() string {
	return fmt.Sprintf("item with sku=%d not found", e.SKU)
}

func NewItemNotFoundError(sku uint32) error {
	return &ItemNotFoundError{SKU: sku}
}
