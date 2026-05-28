package converter

import (
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	loms "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/loms/v1"
)

func ToOrderStatus(orderStatus loms.OrderStatus) entity.OrderStatus {
	switch orderStatus {
	case loms.OrderStatus_ORDER_STATUS_UNSPECIFIED:
		return entity.OrderStatusUnspecified
	case loms.OrderStatus_ORDER_STATUS_NEW:
		return entity.OrderStatusNew
	case loms.OrderStatus_ORDER_STATUS_AWAITING_PAYMENT:
		return entity.OrderStatusAwaitingPayment
	case loms.OrderStatus_ORDER_STATUS_FAILED:
		return entity.OrderStatusFailed
	case loms.OrderStatus_ORDER_STATUS_PAID:
		return entity.OrderStatusPaid
	case loms.OrderStatus_ORDER_STATUS_CANCELLED:
		return entity.OrderStatusCancelled
	default:
		return entity.OrderStatusUnspecified
	}
}

func FromOrderStatus(orderStatus entity.OrderStatus) loms.OrderStatus {
	switch orderStatus {
	case entity.OrderStatusUnspecified:
		return loms.OrderStatus_ORDER_STATUS_UNSPECIFIED
	case entity.OrderStatusNew:
		return loms.OrderStatus_ORDER_STATUS_NEW
	case entity.OrderStatusAwaitingPayment:
		return loms.OrderStatus_ORDER_STATUS_AWAITING_PAYMENT
	case entity.OrderStatusFailed:
		return loms.OrderStatus_ORDER_STATUS_FAILED
	case entity.OrderStatusPaid:
		return loms.OrderStatus_ORDER_STATUS_PAID
	case entity.OrderStatusCancelled:
		return loms.OrderStatus_ORDER_STATUS_CANCELLED
	default:
		return loms.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}
