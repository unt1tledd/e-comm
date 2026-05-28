package product

import (
	"context"
	"errors"
	"testing"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
)

func TestPostgresRepository_CreateProduct(t *testing.T) {
	ctx := context.Background()

	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer db.Close()

	repo := NewPostgresReposytory(db)

	db.ExpectQuery(`INSERT INTO loms\.products`).
		WithArgs("iphone", int64(1000)).
		WillReturnRows(
			pgxmock.NewRows([]string{"sku"}).
				AddRow(int64(123)),
		)

	sku, err := repo.CreateProduct(ctx, &entity.ProductInfo{
		Name:  "iphone",
		Price: 1000,
	})

	require.NoError(t, err)
	require.Equal(t, uint32(123), sku)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestPostgresRepository_CreateProduct_Error(t *testing.T) {
	ctx := context.Background()

	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer db.Close()

	repo := NewPostgresReposytory(db)

	db.ExpectQuery(`INSERT INTO loms\.products`).
		WithArgs("iphone", int64(1000)).
		WillReturnError(errors.New("db error"))

	sku, err := repo.CreateProduct(ctx, &entity.ProductInfo{
		Name:  "iphone",
		Price: 1000,
	})

	require.Error(t, err)
	require.Zero(t, sku)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestPostgresRepository_GetProduct(t *testing.T) {
	ctx := context.Background()

	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer db.Close()

	repo := NewPostgresReposytory(db)

	db.ExpectQuery(`SELECT (.+) FROM loms\.products`).
		WithArgs(int64(123)).
		WillReturnRows(
			pgxmock.NewRows([]string{"sku", "name", "price"}).
				AddRow(int64(123), "iphone", int64(1000)),
		)

	product, err := repo.GetProduct(ctx, 123)

	require.NoError(t, err)
	require.Equal(t, &entity.ProductInfo{
		Name:  "iphone",
		Price: 1000,
	}, product)

	require.NoError(t, db.ExpectationsWereMet())
}

func TestPostgresRepository_GetProduct_Error(t *testing.T) {
	ctx := context.Background()

	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer db.Close()

	repo := NewPostgresReposytory(db)

	expectedErr := errors.New("db error")

	db.ExpectQuery(`SELECT (.+) FROM loms\.products`).
		WithArgs(int64(123)).
		WillReturnError(expectedErr)

	product, err := repo.GetProduct(ctx, 123)

	require.Nil(t, product)
	require.ErrorIs(t, err, expectedErr)
	require.NoError(t, db.ExpectationsWereMet())
}
