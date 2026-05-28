package order

import (
	"context"
	"errors"
	"fmt"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	xerror "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	sqlc "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/repository/order/sqlc"
	"github.com/jackc/pgx/v5"

	"github.com/igoroutine-courses/microservices.ecommerce.pkg/transactor"
)

type (
	DB interface {
		Begin(ctx context.Context) (pgx.Tx, error)
		sqlc.DBTX
	}
)

type postgresRepository struct {
	queries *sqlc.Queries
	db      DB
}

func NewPostgresReposytory(db DB) *postgresRepository {
	return &postgresRepository{
		queries: sqlc.New(db),
		db:      db,
	}
}

func (r *postgresRepository) getQueries(ctx context.Context) *sqlc.Queries {
	if tx, err := transactor.ExtractTx(ctx); err == nil {
		return sqlc.New(tx)
	}

	return r.queries
}

func (r *postgresRepository) CreateOrder(ctx context.Context, o *entity.Order) (int64, error) {
	queries := r.getQueries(ctx)

	orderID, err := queries.CreateOrder(ctx, o.UserID)

	if err != nil {
		return 0, fmt.Errorf("insert order: %w", err)
	}

	for _, item := range o.Items {
		if err = queries.CreateOrderItem(ctx, sqlc.CreateOrderItemParams{
			OrderID: orderID,
			Sku:     int64(item.Sku),
			Count:   int64(item.Count),
		}); err != nil {
			return 0, fmt.Errorf("insert order item sku=%d: %w", item.Sku, err)
		}
	}

	return orderID, nil
}

func (r *postgresRepository) GetOrder(ctx context.Context, orderID int64) (*entity.Order, error) {
	queries := r.getQueries(ctx)

	orderRow, err := queries.GetOrder(ctx, orderID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, xerror.ErrOrderNotFound
		}

		return nil, fmt.Errorf("get order: %w", err)
	}

	orderItems, err := queries.GetOrderItems(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("get order items: %w", err)
	}

	items := make([]entity.Item, len(orderItems))
	for i, it := range orderItems {
		items[i] = entity.Item{
			Sku:   uint32(it.Sku),
			Count: uint32(it.Count),
		}
	}

	order := &entity.Order{
		ID:        orderRow.ID,
		UserID:    orderRow.UserID,
		Items:     items,
		Status:    toEntityStatus(orderRow.Status),
		CreatedAt: orderRow.CreatedAt.Time,
		UpdatedAt: orderRow.UpdatedAt.Time,
	}

	return order, nil
}

func (r *postgresRepository) GetOrderForUpdate(ctx context.Context, orderID int64) (*entity.Order, error) {
	queries := r.getQueries(ctx)

	orderRow, err := queries.GetOrderForUpdateByID(ctx, orderID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, xerror.ErrOrderNotFound
		}

		return nil, err
	}

	itemRows, err := queries.GetOrderItems(ctx, orderID)

	if err != nil {
		return nil, err
	}

	items := make([]entity.Item, len(itemRows))
	for i, item := range itemRows {
		items[i] = entity.Item{
			Sku:   uint32(item.Sku),
			Count: uint32(item.Count),
		}
	}

	order := &entity.Order{
		ID:        orderRow.ID,
		UserID:    orderRow.UserID,
		Status:    toEntityStatus(orderRow.Status),
		Items:     items,
		CreatedAt: orderRow.CreatedAt.Time,
		UpdatedAt: orderRow.UpdatedAt.Time,
	}

	return order, nil
}

func (r *postgresRepository) UpdateStatusOrder(ctx context.Context, orderID int64, status entity.OrderStatus) error {
	queries := r.getQueries(ctx)

	cntRows, err := queries.UpdateOrderStatus(ctx, sqlc.UpdateOrderStatusParams{
		ID:     orderID,
		Status: toSQLCStatus(status),
	})

	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}

	if cntRows == 0 {
		return xerror.ErrOrderNotFound
	}

	return nil
}

func toEntityStatus(status sqlc.LomsOrderStatus) entity.OrderStatus {
	switch status {
	case sqlc.LomsOrderStatusNew:
		return entity.OrderStatusNew
	case sqlc.LomsOrderStatusAwaitingpayment:
		return entity.OrderStatusAwaitingPayment
	case sqlc.LomsOrderStatusFailed:
		return entity.OrderStatusFailed
	case sqlc.LomsOrderStatusPaid:
		return entity.OrderStatusPaid
	case sqlc.LomsOrderStatusCancelled:
		return entity.OrderStatusCancelled
	default:
		return entity.OrderStatusNew
	}
}

func toSQLCStatus(status entity.OrderStatus) sqlc.LomsOrderStatus {
	switch status {
	case entity.OrderStatusNew:
		return sqlc.LomsOrderStatusNew
	case entity.OrderStatusAwaitingPayment:
		return sqlc.LomsOrderStatusAwaitingpayment
	case entity.OrderStatusFailed:
		return sqlc.LomsOrderStatusFailed
	case entity.OrderStatusPaid:
		return sqlc.LomsOrderStatusPaid
	case entity.OrderStatusCancelled:
		return sqlc.LomsOrderStatusCancelled
	default:
		return sqlc.LomsOrderStatusNew
	}
}
