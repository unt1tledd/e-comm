package stocks

import (
	"context"
	"errors"
	"testing"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/usercase/stocks/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestStocksService_GetStock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sku       uint32
		count     uint64
		mockSetup func(
			repo *mocks.MockstocksRepository,
			tx *mocks.Mocktransactor,
			sku uint32,
			count uint64,
		)
		want    uint64
		wantErr error
	}{
		{
			name:  "success",
			sku:   1001,
			count: 15,
			mockSetup: func(
				repo *mocks.MockstocksRepository,
				tx *mocks.Mocktransactor,
				sku uint32,
				count uint64,
			) {
				tx.EXPECT().
					WithTx(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					})

				repo.EXPECT().
					GetStock(gomock.Any(), sku).
					Return(count, nil)
			},
			want: 15,
		},
		{
			name:  "repository error",
			sku:   1002,
			count: 0,
			mockSetup: func(
				repo *mocks.MockstocksRepository,
				tx *mocks.Mocktransactor,
				sku uint32,
				count uint64,
			) {
				errRepo := errors.New("repo error")

				tx.EXPECT().
					WithTx(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					})

				repo.EXPECT().
					GetStock(gomock.Any(), sku).
					Return(uint64(0), errRepo)
			},
			want:    0,
			wantErr: errors.New("repo error"),
		},
		{
			name:  "transaction error",
			sku:   1003,
			count: 0,
			mockSetup: func(
				repo *mocks.MockstocksRepository,
				tx *mocks.Mocktransactor,
				sku uint32,
				count uint64,
			) {
				tx.EXPECT().
					WithTx(gomock.Any(), gomock.Any()).
					Return(errors.New("tx error"))
			},
			want:    0,
			wantErr: errors.New("tx error"),
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			repo := mocks.NewMockstocksRepository(ctrl)
			tx := mocks.NewMocktransactor(ctrl)

			tt.mockSetup(repo, tx, tt.sku, tt.count)

			service := NewStocksService(repo, tx)

			got, err := service.GetStock(context.Background(), tt.sku)

			if tt.wantErr != nil {
				require.Error(t, err)
				require.EqualError(t, err, tt.wantErr.Error())
				require.Equal(t, tt.want, got)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestStocksService_SetStock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sku       uint32
		count     uint64
		mockSetup func(
			repo *mocks.MockstocksRepository,
			tx *mocks.Mocktransactor,
			sku uint32,
			count uint64,
		)
		wantErr error
	}{
		{
			name:  "success",
			sku:   1001,
			count: 20,
			mockSetup: func(
				repo *mocks.MockstocksRepository,
				tx *mocks.Mocktransactor,
				sku uint32,
				count uint64,
			) {
				tx.EXPECT().
					WithTx(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					})

				repo.EXPECT().
					SetStock(gomock.Any(), sku, count).
					Return(nil)
			},
		},
		{
			name:  "repository error",
			sku:   1002,
			count: 30,
			mockSetup: func(
				repo *mocks.MockstocksRepository,
				tx *mocks.Mocktransactor,
				sku uint32,
				count uint64,
			) {
				errRepo := errors.New("repo error")

				tx.EXPECT().
					WithTx(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					})

				repo.EXPECT().
					SetStock(gomock.Any(), sku, count).
					Return(errRepo)
			},
			wantErr: errors.New("repo error"),
		},
		{
			name:  "transaction error",
			sku:   1003,
			count: 40,
			mockSetup: func(
				repo *mocks.MockstocksRepository,
				tx *mocks.Mocktransactor,
				sku uint32,
				count uint64,
			) {
				tx.EXPECT().
					WithTx(gomock.Any(), gomock.Any()).
					Return(errors.New("tx error"))
			},
			wantErr: errors.New("tx error"),
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			repo := mocks.NewMockstocksRepository(ctrl)
			tx := mocks.NewMocktransactor(ctrl)

			tt.mockSetup(repo, tx, tt.sku, tt.count)

			service := NewStocksService(repo, tx)

			err := service.SetStock(context.Background(), tt.sku, tt.count)

			if tt.wantErr != nil {
				require.Error(t, err)
				require.EqualError(t, err, tt.wantErr.Error())
				return
			}

			require.NoError(t, err)
		})
	}
}
