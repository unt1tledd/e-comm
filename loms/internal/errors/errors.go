package errors

import (
	"errors"
	"fmt"
)

var (
	ErrOrderNotFound     = errors.New("order not found")
	ErrStockNotFound     = errors.New("stock not found")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrInvalidStatus     = errors.New("invalid status")
	ErrNotFoundProduct   = errors.New("not found product")
)

type OutboxStatusUpdateError struct {
	Expected int
	Actual   int64
}

func (e OutboxStatusUpdateError) Error() string {
	return fmt.Sprintf("outbox status partial update: expected %d rows, got %d", e.Expected, e.Actual)
}
