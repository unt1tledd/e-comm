package stocks

import (
	"context"
	"errors"
	"testing"

	xerror "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
)

func TestPostgresRepository_CreateStock(t *testing.T) {
	ctx := context.Background()

	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer db.Close()

	repo := NewPostgresReposytory(db)

	db.ExpectExec(`INSERT INTO loms\.available_stocks`).
		WithArgs(int64(123)).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = repo.CreateStock(ctx, 123)

	require.NoError(t, err)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestPostgresRepository_CreateStock_Error(t *testing.T) {
	ctx := context.Background()

	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer db.Close()

	repo := NewPostgresReposytory(db)

	expectedErr := errors.New("db error")

	db.ExpectExec(`INSERT INTO loms\.available_stocks`).
		WithArgs(int64(123)).
		WillReturnError(expectedErr)

	err = repo.CreateStock(ctx, 123)

	require.ErrorIs(t, err, expectedErr)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestPostgresRepository_ReserveStock(t *testing.T) {
	ctx := context.Background()

	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer db.Close()

	repo := NewPostgresReposytory(db)

	db.ExpectExec(`UPDATE loms\.available_stocks`).
		WithArgs(int64(123), int64(5)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.ReserveStock(ctx, 123, 5)

	require.NoError(t, err)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestPostgresRepository_ReserveStock_InsufficientStock(t *testing.T) {
	ctx := context.Background()

	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer db.Close()

	repo := NewPostgresReposytory(db)

	db.ExpectExec(`UPDATE loms\.available_stocks`).
		WithArgs(int64(123), int64(5)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	err = repo.ReserveStock(ctx, 123, 5)

	require.ErrorIs(t, err, xerror.ErrInsufficientStock)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestPostgresRepository_ReserveStock_Error(t *testing.T) {
	ctx := context.Background()

	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer db.Close()

	repo := NewPostgresReposytory(db)

	expectedErr := errors.New("db error")

	db.ExpectExec(`UPDATE loms\.available_stocks`).
		WithArgs(int64(123), int64(5)).
		WillReturnError(expectedErr)

	err = repo.ReserveStock(ctx, 123, 5)

	require.Error(t, err)
	require.ErrorIs(t, err, expectedErr)
	require.Contains(t, err.Error(), "decrement available stock")
	require.NoError(t, db.ExpectationsWereMet())
}

func TestPostgresRepository_ReleaseStock(t *testing.T) {
	ctx := context.Background()

	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer db.Close()

	repo := NewPostgresReposytory(db)

	db.ExpectExec(`INSERT INTO loms\.available_stocks`).
		WithArgs(int64(123), int64(5)).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = repo.ReleaseStock(ctx, 123, 5)

	require.NoError(t, err)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestPostgresRepository_ReleaseStock_Error(t *testing.T) {
	ctx := context.Background()

	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer db.Close()

	repo := NewPostgresReposytory(db)

	expectedErr := errors.New("db error")

	db.ExpectExec(`INSERT INTO loms\.available_stocks`).
		WithArgs(int64(123), int64(5)).
		WillReturnError(expectedErr)

	err = repo.ReleaseStock(ctx, 123, 5)

	require.ErrorIs(t, err, expectedErr)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestPostgresRepository_SetStock(t *testing.T) {
	ctx := context.Background()

	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer db.Close()

	repo := NewPostgresReposytory(db)

	db.ExpectExec(`INSERT INTO loms\.available_stocks`).
		WithArgs(int64(123), int64(10)).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = repo.SetStock(ctx, 123, 10)

	require.NoError(t, err)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestPostgresRepository_SetStock_Error(t *testing.T) {
	ctx := context.Background()

	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer db.Close()

	repo := NewPostgresReposytory(db)

	expectedErr := errors.New("db error")

	db.ExpectExec(`INSERT INTO loms\.available_stocks`).
		WithArgs(int64(123), int64(10)).
		WillReturnError(expectedErr)

	err = repo.SetStock(ctx, 123, 10)

	require.ErrorIs(t, err, expectedErr)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestPostgresRepository_GetStock(t *testing.T) {
	ctx := context.Background()

	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer db.Close()

	repo := NewPostgresReposytory(db)

	db.ExpectQuery(`SELECT (.+) FROM loms\.available_stocks`).
		WithArgs(int64(123)).
		WillReturnRows(
			pgxmock.NewRows([]string{"count"}).
				AddRow(int64(10)),
		)

	count, err := repo.GetStock(ctx, 123)

	require.NoError(t, err)
	require.Equal(t, uint64(10), count)
	require.NoError(t, db.ExpectationsWereMet())
}

func TestPostgresRepository_GetStock_Error(t *testing.T) {
	ctx := context.Background()

	db, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer db.Close()

	repo := NewPostgresReposytory(db)

	expectedErr := errors.New("db error")

	db.ExpectQuery(`SELECT (.+) FROM loms\.available_stocks`).
		WithArgs(int64(123)).
		WillReturnError(expectedErr)

	count, err := repo.GetStock(ctx, 123)

	require.ErrorIs(t, err, expectedErr)
	require.Zero(t, count)
	require.NoError(t, db.ExpectationsWereMet())
}
