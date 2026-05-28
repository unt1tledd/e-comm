package product

import (
	"context"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	sqlc "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/repository/product/sqlc"
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

func (r *postgresRepository) CreateProduct(ctx context.Context, product *entity.ProductInfo) (uint32, error) {
	queries := r.getQueries(ctx)

	sku, err := queries.CreateProduct(ctx, sqlc.CreateProductParams{
		Name:  product.Name,
		Price: int64(product.Price),
	})

	return uint32(sku), err
}

func (r *postgresRepository) GetProduct(ctx context.Context, sku uint32) (*entity.ProductInfo, error) {
	queries := r.getQueries(ctx)

	lomsProduct, err := queries.GetProduct(ctx, int64(sku))
	if err != nil {
		return nil, err
	}

	return &entity.ProductInfo{
		Name:  lomsProduct.Name,
		Price: uint32(lomsProduct.Price),
	}, nil
}
