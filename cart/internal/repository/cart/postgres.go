package cart

import (
	"context"

	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/entity"
	sqlc "github.com/igoroutine-courses/microservices.ecommerce.cart/internal/repository/cart/sqlc"
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

func (r *postgresRepository) AddItem(ctx context.Context, userID int64, item entity.OrderItem) error {
	queries := r.getQueries(ctx)

	err := queries.AddCartItem(ctx, sqlc.AddCartItemParams{
		UserID: userID,
		Sku:    int64(item.Sku),
		Count:  int64(item.Count),
	})

	return err
}

func (r *postgresRepository) DeleteItem(ctx context.Context, userID int64, sku uint32) error {
	queries := r.getQueries(ctx)

	return queries.DeleteCartItem(ctx, sqlc.DeleteCartItemParams{
		UserID: userID,
		Sku:    int64(sku),
	})
}

func (r *postgresRepository) GetItemsByUserID(ctx context.Context, userID int64) ([]entity.OrderItem, error) {
	queries := r.getQueries(ctx)

	items, err := queries.GetCartItems(ctx, userID)
	if err != nil {
		return nil, err
	}
	return toOrderItemEntity(items), nil
}

func (r *postgresRepository) ClearCart(ctx context.Context, userID int64) error {
	queries := r.getQueries(ctx)

	return queries.ClearCart(ctx, userID)
}

func toOrderItemEntity(items []sqlc.GetCartItemsRow) []entity.OrderItem {
	orderItems := make([]entity.OrderItem, len(items))
	for i, it := range items {
		orderItems[i] = entity.OrderItem{
			Sku:   uint32(it.Sku),
			Count: uint32(it.Count),
		}
	}

	return orderItems
}
