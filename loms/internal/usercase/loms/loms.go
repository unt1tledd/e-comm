package loms

import (
	"context"
	"encoding/json"
	"strconv"

	notifications "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/adapter/notifications/converter"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	xerror "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/port"
)

//go:generate mockgen -source=loms.go -destination=mocks/loms_mocks.go -package=mocks

type (
	orderRepository interface {
		CreateOrder(ctx context.Context, order *entity.Order) (int64, error)
		GetOrder(ctx context.Context, orderID int64) (*entity.Order, error)
		GetOrderForUpdate(ctx context.Context, orderID int64) (*entity.Order, error)
		UpdateStatusOrder(ctx context.Context, orderID int64, status entity.OrderStatus) error
	}

	stocksRepository interface {
		ReserveStock(ctx context.Context, sku uint32, count uint64) error
		ReleaseStock(ctx context.Context, sku uint32, count uint64) error
	}

	productRepository interface {
		GetProduct(ctx context.Context, sku uint32) (*entity.ProductInfo, error)
	}

	outboxRepository interface {
		SendOutboxMessage(ctx context.Context, message *entity.OutboxMessage) error
	}

	notificationsClient interface {
		SendOrderStatusChangedNotification(ctx context.Context, userID, orderID int64, status port.OrderStatus) error
	}

	transactor interface {
		WithTx(ctx context.Context, f func(ctx context.Context) error) error
	}
)

type lomsService struct {
	orderRepository     orderRepository
	stocksRepository    stocksRepository
	productRepository   productRepository
	outboxRepository    outboxRepository
	notificationsClient notificationsClient
	transactor          transactor
}

func NewLomsService(
	orderRepository orderRepository,
	stocksRepository stocksRepository,
	productRepository productRepository,
	outboxRepository outboxRepository,
	notificationsClient notificationsClient,
	transactor transactor,
) *lomsService {
	return &lomsService{
		orderRepository:     orderRepository,
		stocksRepository:    stocksRepository,
		productRepository:   productRepository,
		outboxRepository:    outboxRepository,
		notificationsClient: notificationsClient,
		transactor:          transactor,
	}
}

func (l *lomsService) CreateOrder(ctx context.Context, userID int64, items []entity.Item) (int64, error) {
	var orderID int64

	err := l.transactor.WithTx(ctx, func(ctx context.Context) error {
		for _, item := range items {
			if err := l.stocksRepository.ReserveStock(ctx, item.Sku, uint64(item.Count)); err != nil {
				return err
			}
		}

		order := &entity.Order{
			UserID: userID,
			Items:  items,
			Status: entity.OrderStatusAwaitingPayment,
		}

		id, err := l.orderRepository.CreateOrder(ctx, order)
		if err != nil {
			return err
		}

		orderID = id
		order.ID = id

		return l.createOutboxOrderStatusChangedNotification(ctx, order)
	})
	if err != nil {
		return 0, err
	}

	return orderID, nil
}

func (l *lomsService) GetOrder(ctx context.Context, orderID int64) (*entity.Order, error) {
	return l.orderRepository.GetOrder(ctx, orderID)
}

func (l *lomsService) PayOrder(ctx context.Context, orderID int64) error {
	return l.transactor.WithTx(ctx, func(ctx context.Context) error {
		order, err := l.orderRepository.GetOrderForUpdate(ctx, orderID)
		if err != nil {
			return err
		}

		if order.Status != entity.OrderStatusAwaitingPayment {
			return xerror.ErrInvalidStatus
		}

		if err := l.orderRepository.UpdateStatusOrder(ctx, orderID, entity.OrderStatusPaid); err != nil {
			return err
		}

		order.Status = entity.OrderStatusPaid

		return l.createOutboxOrderStatusChangedNotification(ctx, order)
	})
}

func (l *lomsService) CancelOrder(ctx context.Context, orderID int64) error {
	return l.transactor.WithTx(ctx, func(ctx context.Context) error {
		order, err := l.orderRepository.GetOrderForUpdate(ctx, orderID)
		if err != nil {
			return err
		}

		if order.Status != entity.OrderStatusAwaitingPayment {
			return xerror.ErrInvalidStatus
		}

		for _, item := range order.Items {
			if err := l.stocksRepository.ReleaseStock(ctx, item.Sku, uint64(item.Count)); err != nil {
				return err
			}
		}

		if err := l.orderRepository.UpdateStatusOrder(ctx, orderID, entity.OrderStatusCancelled); err != nil {
			return err
		}

		order.Status = entity.OrderStatusCancelled

		return l.createOutboxOrderStatusChangedNotification(ctx, order)
	})
}

func (l *lomsService) OrderStatusChangedNotificationKindHandler(ctx context.Context, data []byte) error {
	var body port.OrderStatusChangedNotification
	if err := json.Unmarshal(data, &body); err != nil {
		return err
	}

	return l.notificationsClient.SendOrderStatusChangedNotification(
		ctx,
		body.UserID,
		body.OrderID,
		body.Status,
	)
}

func (l *lomsService) createOutboxOrderStatusChangedNotification(ctx context.Context, order *entity.Order) error {
	body := port.OrderStatusChangedNotification{
		OrderID: order.ID,
		UserID:  order.UserID,
		Status:  notifications.FromOrderStatus(order.Status),
	}

	rawBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	message := &entity.OutboxMessage{
		IdempotencyKey: l.createOutboxIdempotencyKey(order.ID, order.Status),
		Kind:           entity.KindNotification,
		Data:           rawBody,
	}

	return l.outboxRepository.SendOutboxMessage(ctx, message)
}

func (l *lomsService) createOutboxIdempotencyKey(orderID int64, status entity.OrderStatus) string {
	return strconv.FormatInt(orderID, 10) + "_" + strconv.Itoa(int(status))
}
