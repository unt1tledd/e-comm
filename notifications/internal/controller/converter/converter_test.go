package converter

import (
	"testing"

	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/entity"
	notificationspb "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/notifications/api/v1"
	"github.com/stretchr/testify/require"
)

func TestToOrderStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   notificationspb.OrderStatus
		want entity.OrderStatus
	}{
		{
			name: "new",
			in:   notificationspb.OrderStatus_ORDER_STATUS_NEW,
			want: entity.OrderStatusNew,
		},
		{
			name: "awaiting payment",
			in:   notificationspb.OrderStatus_ORDER_STATUS_AWAITING_PAYMENT,
			want: entity.OrderStatusAwaitingPayment,
		},
		{
			name: "failed",
			in:   notificationspb.OrderStatus_ORDER_STATUS_FAILED,
			want: entity.OrderStatusFailed,
		},
		{
			name: "paid",
			in:   notificationspb.OrderStatus_ORDER_STATUS_PAID,
			want: entity.OrderStatusPaid,
		},
		{
			name: "cancelled",
			in:   notificationspb.OrderStatus_ORDER_STATUS_CANCELLED,
			want: entity.OrderStatusCancelled,
		},
		{
			name: "unknown",
			in:   notificationspb.OrderStatus_ORDER_STATUS_UNSPECIFIED,
			want: entity.OrderStatusUnspecified,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ToOrderStatus(tt.in)
			require.Equal(t, tt.want, got)
		})
	}
}
