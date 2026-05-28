package notifications

import (
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/port"
	notifications "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/notifications/api/v1"
)

func FromOrderStatus(orderStatus port.OrderStatus) notifications.OrderStatus {
	switch orderStatus {
	case port.OrderStatusNew:
		return notifications.OrderStatus_ORDER_STATUS_NEW
	case port.OrderStatusAwaitingPayment:
		return notifications.OrderStatus_ORDER_STATUS_AWAITING_PAYMENT
	case port.OrderStatusFailed:
		return notifications.OrderStatus_ORDER_STATUS_FAILED
	case port.OrderStatusPaid:
		return notifications.OrderStatus_ORDER_STATUS_PAID
	case port.OrderStatusCancelled:
		return notifications.OrderStatus_ORDER_STATUS_CANCELLED
	default:
		return notifications.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}
