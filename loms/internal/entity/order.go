package entity

import "time"

type Order struct {
	ID        int64
	UserID    int64
	Items     []Item
	Status    OrderStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

type OrderStatus int

const (
	OrderStatusUnspecified OrderStatus = iota
	OrderStatusNew
	OrderStatusAwaitingPayment
	OrderStatusFailed
	OrderStatusPaid
	OrderStatusCancelled
)
