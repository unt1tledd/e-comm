package order

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	xerror "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	sqlc "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/repository/order/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
)

func TestPostgresRepository_CreateOrder(t *testing.T) {
	ctx := context.Background()

	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer db.Close()

	repo := NewPostgresReposytory(db)

	db.ExpectQuery(`INSERT INTO loms\.orders`).
		WithArgs(int64(10)).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(100)))

	db.ExpectExec(`INSERT INTO loms\.order_info`).
		WithArgs(int64(100), int64(111), int64(2)).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	db.ExpectExec(`INSERT INTO loms\.order_info`).
		WithArgs(int64(100), int64(222), int64(3)).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	id, err := repo.CreateOrder(ctx, &entity.Order{
		UserID: 10,
		Items: []entity.Item{
			{Sku: 111, Count: 2},
			{Sku: 222, Count: 3},
		},
	})

	require.NoError(t, err)
	require.Equal(t, int64(100), id)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestPostgresRepository_CreateOrder_Error(t *testing.T) {
	ctx := context.Background()

	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer db.Close()

	repo := NewPostgresReposytory(db)

	db.ExpectQuery(`INSERT INTO loms\.orders`).
		WithArgs(int64(10)).
		WillReturnError(errors.New("db error"))

	id, err := repo.CreateOrder(ctx, &entity.Order{
		UserID: 10,
		Items:  []entity.Item{{Sku: 111, Count: 2}},
	})

	require.Error(t, err)
	require.Zero(t, id)
	require.Contains(t, err.Error(), "insert order")
	require.NoError(t, db.ExpectationsWereMet())
}

func TestPostgresRepository_GetOrder(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer db.Close()

	repo := NewPostgresReposytory(db)

	db.ExpectQuery(`SELECT (.+) FROM loms\.orders`).
		WithArgs(int64(100)).
		WillReturnRows(
			pgxmock.NewRows([]string{
				"id",
				"user_id",
				"status",
				"created_at",
				"updated_at",
			}).AddRow(
				int64(100),
				int64(10),
				sqlc.LomsOrderStatusAwaitingpayment,
				now,
				now,
			),
		)

	db.ExpectQuery(`SELECT (.+) FROM loms\.order_info`).
		WithArgs(int64(100)).
		WillReturnRows(
			pgxmock.NewRows([]string{"sku", "count"}).
				AddRow(int64(111), int64(2)).
				AddRow(int64(222), int64(3)),
		)

	order, err := repo.GetOrder(ctx, 100)

	require.NoError(t, err)
	require.Equal(t, int64(100), order.ID)
	require.Equal(t, int64(10), order.UserID)
	require.Equal(t, entity.OrderStatusAwaitingPayment, order.Status)
	require.Equal(t, []entity.Item{
		{Sku: 111, Count: 2},
		{Sku: 222, Count: 3},
	}, order.Items)

	require.NoError(t, db.ExpectationsWereMet())
}

func TestPostgresRepository_GetOrder_NotFound(t *testing.T) {
	ctx := context.Background()

	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer db.Close()

	repo := NewPostgresReposytory(db)

	db.ExpectQuery(`SELECT (.+) FROM loms\.orders`).
		WithArgs(int64(404)).
		WillReturnError(pgx.ErrNoRows)

	order, err := repo.GetOrder(ctx, 404)

	require.Nil(t, order)
	require.ErrorIs(t, err, xerror.ErrOrderNotFound)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestPostgresRepository_GetOrderForUpdate(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer db.Close()

	repo := NewPostgresReposytory(db)

	db.ExpectQuery(`SELECT (.+) FROM loms\.orders(.+)FOR UPDATE`).
		WithArgs(int64(100)).
		WillReturnRows(
			pgxmock.NewRows([]string{
				"id",
				"user_id",
				"status",
				"created_at",
				"updated_at",
			}).AddRow(
				int64(100),
				int64(10),
				sqlc.LomsOrderStatusPaid,
				now,
				now,
			),
		)

	db.ExpectQuery(`SELECT (.+) FROM loms\.order_info`).
		WithArgs(int64(100)).
		WillReturnRows(
			pgxmock.NewRows([]string{"sku", "count"}).
				AddRow(int64(111), int64(2)),
		)

	order, err := repo.GetOrderForUpdate(ctx, 100)

	require.NoError(t, err)
	require.Equal(t, int64(100), order.ID)
	require.Equal(t, entity.OrderStatusPaid, order.Status)
	require.Equal(t, []entity.Item{
		{Sku: 111, Count: 2},
	}, order.Items)

	require.NoError(t, db.ExpectationsWereMet())
}

func TestPostgresRepository_UpdateStatusOrder(t *testing.T) {
	ctx := context.Background()

	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer db.Close()

	repo := NewPostgresReposytory(db)

	db.ExpectExec(`UPDATE loms\.orders`).
		WithArgs(int64(100), sqlc.LomsOrderStatusPaid).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.UpdateStatusOrder(ctx, 100, entity.OrderStatusPaid)

	require.NoError(t, err)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestPostgresRepository_UpdateStatusOrder_NotFound(t *testing.T) {
	ctx := context.Background()

	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer db.Close()

	repo := NewPostgresReposytory(db)

	db.ExpectExec(`UPDATE loms\.orders`).
		WithArgs(int64(404), sqlc.LomsOrderStatusCancelled).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	err = repo.UpdateStatusOrder(ctx, 404, entity.OrderStatusCancelled)

	require.ErrorIs(t, err, xerror.ErrOrderNotFound)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestStatusConverters(t *testing.T) {
	require.Equal(t, entity.OrderStatusNew, toEntityStatus(sqlc.LomsOrderStatusNew))
	require.Equal(t, entity.OrderStatusAwaitingPayment, toEntityStatus(sqlc.LomsOrderStatusAwaitingpayment))
	require.Equal(t, entity.OrderStatusFailed, toEntityStatus(sqlc.LomsOrderStatusFailed))
	require.Equal(t, entity.OrderStatusPaid, toEntityStatus(sqlc.LomsOrderStatusPaid))
	require.Equal(t, entity.OrderStatusCancelled, toEntityStatus(sqlc.LomsOrderStatusCancelled))

	require.Equal(t, sqlc.LomsOrderStatusNew, toSQLCStatus(entity.OrderStatusNew))
	require.Equal(t, sqlc.LomsOrderStatusAwaitingpayment, toSQLCStatus(entity.OrderStatusAwaitingPayment))
	require.Equal(t, sqlc.LomsOrderStatusFailed, toSQLCStatus(entity.OrderStatusFailed))
	require.Equal(t, sqlc.LomsOrderStatusPaid, toSQLCStatus(entity.OrderStatusPaid))
	require.Equal(t, sqlc.LomsOrderStatusCancelled, toSQLCStatus(entity.OrderStatusCancelled))
}
