package product

import (
	"context"
	"errors"
	"testing"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	xerror "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/usercase/product/mocks"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestProductService_CreateProduct(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errRepo := errors.New("repo error")
	errStock := errors.New("stock error")

	tests := []struct {
		name      string
		setup     func(*mocks.MockproductRepository, *mocks.MockstocksRepository, *mocks.Mocktransactor)
		wantSKU   uint32
		wantError error
	}{
		{
			name: "success",
			setup: func(pr *mocks.MockproductRepository, sr *mocks.MockstocksRepository, tr *mocks.Mocktransactor) {
				tr.EXPECT().
					WithTx(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					})

				pr.EXPECT().
					CreateProduct(gomock.Any(), &entity.ProductInfo{Name: "iphone", Price: 1000}).
					Return(uint32(123), nil)

				sr.EXPECT().
					CreateStock(gomock.Any(), uint32(123)).
					Return(nil)
			},
			wantSKU: 123,
		},
		{
			name: "create product error",
			setup: func(pr *mocks.MockproductRepository, sr *mocks.MockstocksRepository, tr *mocks.Mocktransactor) {
				tr.EXPECT().
					WithTx(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					})

				pr.EXPECT().
					CreateProduct(gomock.Any(), &entity.ProductInfo{Name: "iphone", Price: 1000}).
					Return(uint32(0), errRepo)
			},
			wantError: errRepo,
		},
		{
			name: "create stock error",
			setup: func(pr *mocks.MockproductRepository, sr *mocks.MockstocksRepository, tr *mocks.Mocktransactor) {
				tr.EXPECT().
					WithTx(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					})

				pr.EXPECT().
					CreateProduct(gomock.Any(), &entity.ProductInfo{Name: "iphone", Price: 1000}).
					Return(uint32(123), nil)

				sr.EXPECT().
					CreateStock(gomock.Any(), uint32(123)).
					Return(errStock)
			},
			wantError: errStock,
		},
		{
			name: "transaction error",
			setup: func(pr *mocks.MockproductRepository, sr *mocks.MockstocksRepository, tr *mocks.Mocktransactor) {
				tr.EXPECT().
					WithTx(gomock.Any(), gomock.Any()).
					Return(errRepo)
			},
			wantError: errRepo,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			pr := mocks.NewMockproductRepository(ctrl)
			sr := mocks.NewMockstocksRepository(ctrl)
			tr := mocks.NewMocktransactor(ctrl)

			tt.setup(pr, sr, tr)

			service := NewProductService(pr, sr, tr)

			gotSKU, err := service.CreateProduct(ctx, "iphone", 1000)

			require.ErrorIs(t, err, tt.wantError)
			require.Equal(t, tt.wantSKU, gotSKU)
		})
	}
}

func TestProductService_GetProduct(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	errRepo := errors.New("repo error")

	tests := []struct {
		name        string
		setup       func(*mocks.MockproductRepository)
		wantProduct *entity.ProductInfo
		wantError   error
	}{
		{
			name: "success",
			setup: func(pr *mocks.MockproductRepository) {
				pr.EXPECT().
					GetProduct(gomock.Any(), uint32(123)).
					Return(&entity.ProductInfo{
						Name:  "iphone",
						Price: 1000,
					}, nil)
			},
			wantProduct: &entity.ProductInfo{
				Name:  "iphone",
				Price: 1000,
			},
		},
		{
			name: "not found",
			setup: func(pr *mocks.MockproductRepository) {
				pr.EXPECT().
					GetProduct(gomock.Any(), uint32(123)).
					Return(nil, errRepo)
			},
			wantError: xerror.ErrNotFoundProduct,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			pr := mocks.NewMockproductRepository(ctrl)
			sr := mocks.NewMockstocksRepository(ctrl)
			tr := mocks.NewMocktransactor(ctrl)

			tt.setup(pr)

			service := NewProductService(pr, sr, tr)

			gotProduct, err := service.GetProduct(ctx, 123)

			require.ErrorIs(t, err, tt.wantError)
			require.Equal(t, tt.wantProduct, gotProduct)
		})
	}
}
