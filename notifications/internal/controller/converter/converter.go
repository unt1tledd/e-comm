package converter

import (
	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/entity"
	notifications "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/notifications/api/v1"
)

func ToOrderStatus(orderStatus notifications.OrderStatus) entity.OrderStatus {
	switch orderStatus {
	case notifications.OrderStatus_ORDER_STATUS_UNSPECIFIED:
		return entity.OrderStatusUnspecified
	case notifications.OrderStatus_ORDER_STATUS_NEW:
		return entity.OrderStatusNew
	case notifications.OrderStatus_ORDER_STATUS_AWAITING_PAYMENT:
		return entity.OrderStatusAwaitingPayment
	case notifications.OrderStatus_ORDER_STATUS_FAILED:
		return entity.OrderStatusFailed
	case notifications.OrderStatus_ORDER_STATUS_PAID:
		return entity.OrderStatusPaid
	case notifications.OrderStatus_ORDER_STATUS_CANCELLED:
		return entity.OrderStatusCancelled
	default:
		return entity.OrderStatusUnspecified
	}
}
