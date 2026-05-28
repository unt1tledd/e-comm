package stocks

import (
	"context"
	"fmt"

	xerror "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	sqlc "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/repository/stocks/sqlc"
	"github.com/jackc/pgx/v5"

	transactor "github.com/igoroutine-courses/microservices.ecommerce.pkg/transactor"
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

func (r *postgresRepository) CreateStock(ctx context.Context, sku uint32) error {
	queries := r.getQueries(ctx)

	err := queries.CreateStock(ctx, int64(sku))
	return err
}

func (r *postgresRepository) ReserveStock(ctx context.Context, sku uint32, count uint64) error {
	queries := r.getQueries(ctx)

	rowsAffected, err := queries.DecrementAvailableStock(ctx, sqlc.DecrementAvailableStockParams{
		Sku:   int64(sku),
		Count: int64(count),
	})

	if err != nil {
		return fmt.Errorf("decrement available stock: %w", err)
	}

	if rowsAffected == 0 {
		return xerror.ErrInsufficientStock
	}

	return nil
}

func (r *postgresRepository) ReleaseStock(ctx context.Context, sku uint32, count uint64) error {
	return r.getQueries(ctx).AddToAvailableStock(ctx, sqlc.AddToAvailableStockParams{
		Sku:   int64(sku),
		Count: int64(count),
	})
}

func (r *postgresRepository) SetStock(ctx context.Context, sku uint32, count uint64) error {
	return r.getQueries(ctx).UpsertAvailableStock(ctx, sqlc.UpsertAvailableStockParams{
		Sku:   int64(sku),
		Count: int64(count),
	})
}

func (r *postgresRepository) GetStock(ctx context.Context, sku uint32) (uint64, error) {
	amount, err := r.getQueries(ctx).GetAvailableStockAmount(ctx, int64(sku))

	if err != nil {
		return 0, err
	}

	return uint64(amount), nil
}
