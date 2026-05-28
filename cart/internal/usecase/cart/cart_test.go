package cart

import (
	"context"
	"errors"
	"testing"

	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/entity"
	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/port"
	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/usecase/cart/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newTestService(t *testing.T) (
	context.Context,
	*cartService,
	*mocks.MockcartRepository,
	*mocks.MockproductClient,
	*mocks.MocklomsClient,
) {
	t.Helper()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	repo := mocks.NewMockcartRepository(ctrl)
	productClient := mocks.NewMockproductClient(ctrl)
	lomsClient := mocks.NewMocklomsClient(ctrl)

	service := NewCartService(repo, productClient, lomsClient)

	return context.Background(), service, repo, productClient, lomsClient
}

func TestCartService_CheckoutCart(t *testing.T) {
	t.Parallel()

	errRepo := errors.New("repo error")
	errLoms := errors.New("loms error")
	errClear := errors.New("clear error")

	tests := []struct {
		name  string
		setup func(
			ctx context.Context,
			repo *mocks.MockcartRepository,
			loms *mocks.MocklomsClient,
		)
		wantOrderID int64
		wantErr     error
	}{
		{
			name: "success",
			setup: func(ctx context.Context, repo *mocks.MockcartRepository, loms *mocks.MocklomsClient) {
				items := []entity.OrderItem{
					{Sku: 1001, Count: 2},
					{Sku: 2002, Count: 3},
				}

				portItems := []port.Item{
					{SKU: 1001, Count: 2},
					{SKU: 2002, Count: 3},
				}

				gomock.InOrder(
					repo.EXPECT().
						GetItemsByUserID(ctx, int64(10)).
						Return(items, nil),

					loms.EXPECT().
						CreateOrder(ctx, int64(10), portItems).
						Return(int64(777), nil),

					repo.EXPECT().
						ClearCart(ctx, int64(10)).
						Return(nil),
				)
			},
			wantOrderID: 777,
			wantErr:     nil,
		},
		{
			name: "get items error",
			setup: func(ctx context.Context, repo *mocks.MockcartRepository, loms *mocks.MocklomsClient) {
				repo.EXPECT().
					GetItemsByUserID(ctx, int64(10)).
					Return(nil, errRepo)
			},
			wantOrderID: 0,
			wantErr:     errRepo,
		},
		{
			name: "create order error",
			setup: func(ctx context.Context, repo *mocks.MockcartRepository, loms *mocks.MocklomsClient) {
				items := []entity.OrderItem{
					{Sku: 1001, Count: 2},
				}

				portItems := []port.Item{
					{SKU: 1001, Count: 2},
				}

				gomock.InOrder(
					repo.EXPECT().
						GetItemsByUserID(ctx, int64(10)).
						Return(items, nil),

					loms.EXPECT().
						CreateOrder(ctx, int64(10), portItems).
						Return(int64(0), errLoms),
				)
			},
			wantOrderID: 0,
			wantErr:     errLoms,
		},
		{
			name: "clear cart error",
			setup: func(ctx context.Context, repo *mocks.MockcartRepository, loms *mocks.MocklomsClient) {
				items := []entity.OrderItem{
					{Sku: 1001, Count: 2},
				}

				portItems := []port.Item{
					{SKU: 1001, Count: 2},
				}

				gomock.InOrder(
					repo.EXPECT().
						GetItemsByUserID(ctx, int64(10)).
						Return(items, nil),

					loms.EXPECT().
						CreateOrder(ctx, int64(10), portItems).
						Return(int64(777), nil),

					repo.EXPECT().
						ClearCart(ctx, int64(10)).
						Return(errClear),
				)
			},
			wantOrderID: 0,
			wantErr:     errClear,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, service, repo, _, loms := newTestService(t)

			tt.setup(ctx, repo, loms)

			gotOrderID, err := service.CheckoutCart(ctx, 10)

			assert.Equal(t, tt.wantOrderID, gotOrderID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestCartService_ListCart(t *testing.T) {
	t.Parallel()

	errRepo := errors.New("repo error")
	errProduct := errors.New("product not found")

	tests := []struct {
		name  string
		setup func(
			ctx context.Context,
			repo *mocks.MockcartRepository,
			productClient *mocks.MockproductClient,
		)
		wantProducts      []entity.Product
		wantTotalPrice    uint32
		wantNotFoundItems []uint32
		wantErr           error
	}{
		{
			name: "success",
			setup: func(ctx context.Context, repo *mocks.MockcartRepository, productClient *mocks.MockproductClient) {
				items := []entity.OrderItem{
					{Sku: 1001, Count: 2},
					{Sku: 2002, Count: 3},
				}

				repo.EXPECT().
					GetItemsByUserID(ctx, int64(10)).
					Return(items, nil)

				productClient.EXPECT().
					GetProductInfo(ctx, uint32(1001)).
					Return(&port.ProductInfo{
						Name:  "milk",
						Price: 100,
					}, nil)

				productClient.EXPECT().
					GetProductInfo(ctx, uint32(2002)).
					Return(&port.ProductInfo{
						Name:  "bread",
						Price: 50,
					}, nil)
			},
			wantProducts: []entity.Product{
				{Sku: 1001, Count: 2, Name: "milk", Price: 100},
				{Sku: 2002, Count: 3, Name: "bread", Price: 50},
			},
			wantTotalPrice:    350,
			wantNotFoundItems: nil,
			wantErr:           nil,
		},
		{
			name: "get items error",
			setup: func(ctx context.Context, repo *mocks.MockcartRepository, productClient *mocks.MockproductClient) {
				repo.EXPECT().
					GetItemsByUserID(ctx, int64(10)).
					Return(nil, errRepo)
			},
			wantProducts:      nil,
			wantTotalPrice:    0,
			wantNotFoundItems: nil,
			wantErr:           errRepo,
		},
		{
			name: "part of products not found",
			setup: func(ctx context.Context, repo *mocks.MockcartRepository, productClient *mocks.MockproductClient) {
				items := []entity.OrderItem{
					{Sku: 1001, Count: 2},
					{Sku: 2002, Count: 3},
					{Sku: 3003, Count: 1},
				}

				repo.EXPECT().
					GetItemsByUserID(ctx, int64(10)).
					Return(items, nil)

				productClient.EXPECT().
					GetProductInfo(ctx, uint32(1001)).
					Return(&port.ProductInfo{
						Name:  "milk",
						Price: 100,
					}, nil)

				productClient.EXPECT().
					GetProductInfo(ctx, uint32(2002)).
					Return(nil, errProduct)

				productClient.EXPECT().
					GetProductInfo(ctx, uint32(3003)).
					Return(&port.ProductInfo{
						Name:  "cheese",
						Price: 250,
					}, nil)
			},
			wantProducts: []entity.Product{
				{Sku: 1001, Count: 2, Name: "milk", Price: 100},
				{Sku: 3003, Count: 1, Name: "cheese", Price: 250},
			},
			wantTotalPrice:    450,
			wantNotFoundItems: []uint32{2002},
			wantErr:           nil,
		},
		{
			name: "empty cart",
			setup: func(ctx context.Context, repo *mocks.MockcartRepository, productClient *mocks.MockproductClient) {
				repo.EXPECT().
					GetItemsByUserID(ctx, int64(10)).
					Return([]entity.OrderItem{}, nil)
			},
			wantProducts:      []entity.Product{},
			wantTotalPrice:    0,
			wantNotFoundItems: nil,
			wantErr:           nil,
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, service, repo, productClient, _ := newTestService(t)

			tt.setup(ctx, repo, productClient)

			gotCart, err := service.ListCart(ctx, 10)

			assert.Equal(t, tt.wantProducts, gotCart.Items)
			assert.Equal(t, tt.wantTotalPrice, gotCart.TotalPrice)
			assert.Equal(t, tt.wantNotFoundItems, gotCart.NotFoundItems)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestCartService_ClearCart(t *testing.T) {
	t.Parallel()

	errClear := errors.New("clear error")

	tests := []struct {
		name    string
		setup   func(ctx context.Context, repo *mocks.MockcartRepository)
		wantErr error
	}{
		{
			name: "success",
			setup: func(ctx context.Context, repo *mocks.MockcartRepository) {
				repo.EXPECT().
					ClearCart(ctx, int64(10)).
					Return(nil)
			},
			wantErr: nil,
		},
		{
			name: "repository error",
			setup: func(ctx context.Context, repo *mocks.MockcartRepository) {
				repo.EXPECT().
					ClearCart(ctx, int64(10)).
					Return(errClear)
			},
			wantErr: errClear,
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, service, repo, _, _ := newTestService(t)

			tt.setup(ctx, repo)

			err := service.ClearCart(ctx, 10)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
		})
	}
}
