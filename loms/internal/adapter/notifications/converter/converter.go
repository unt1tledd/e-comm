package notifications

import (
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/port"
)

func FromOrderStatus(orderStatus entity.OrderStatus) port.OrderStatus {
	switch orderStatus {
	case entity.OrderStatusNew:
		return port.OrderStatusNew
	case entity.OrderStatusAwaitingPayment:
		return port.OrderStatusAwaitingPayment
	case entity.OrderStatusFailed:
		return port.OrderStatusFailed
	case entity.OrderStatusPaid:
		return port.OrderStatusPaid
	case entity.OrderStatusCancelled:
		return port.OrderStatusCancelled
	default:
		return port.OrderStatusUnspecified
	}
}
