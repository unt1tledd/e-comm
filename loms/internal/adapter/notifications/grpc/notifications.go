package notifications

import (
	"context"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/port"
	notifications "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/notifications/api/v1"
)

type notificationsClient struct {
	client notifications.NotificationsClient
}

func NewNotificationsClient(
	client notifications.NotificationsClient,
) *notificationsClient {
	return &notificationsClient{
		client: client,
	}
}

func (n *notificationsClient) SendOrderStatusChangedNotification(
	ctx context.Context,
	userID,
	orderID int64,
	status port.OrderStatus,
) error {
	_, err := n.client.SendOrderStatusChangedNotification(ctx, &notifications.OrderStatusChangedNotificationRequest{
		UserId:  userID,
		OrderId: orderID,
		Status:  FromOrderStatus(status),
	})

	if err != nil {
		return err
	}

	return nil
}
