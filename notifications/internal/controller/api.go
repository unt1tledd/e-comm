package controller

import (
	notifications "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/notifications/api/v1"
)

type API struct {
	notificationsServer notifications.NotificationsServer
}

func New(
	notificationsServer notifications.NotificationsServer,
) *API {
	return &API{
		notificationsServer: notificationsServer,
	}
}

func (a *API) GetSrv() notifications.NotificationsServer {
	return a.notificationsServer
}
