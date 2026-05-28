package item

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/entity"
	xerror "github.com/igoroutine-courses/microservices.ecommerce.cart/internal/errors"
	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/port"
	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/usecase/item/mocks"
)

func newTestService(t *testing.T) (
	context.Context,
	*itemService,
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

	service := NewItemService(repo, productClient, lomsClient)

	return context.Background(), service, repo, productClient, lomsClient
}

func requireErrorIs(t *testing.T, got, want error) {
	t.Helper()

	if !errors.Is(got, want) {
		t.Fatalf("got error: %v, want: %v", got, want)
	}
}

func TestItemService_AddItem(t *testing.T) {
	t.Parallel()

	errProductClient := errors.New("product client error")
	errLomsClient := errors.New("loms client error")
	errRepository := errors.New("repository error")

	tests := []struct {
		name  string
		setup func(
			ctx context.Context,
			repo *mocks.MockcartRepository,
			productClient *mocks.MockproductClient,
			lomsClient *mocks.MocklomsClient,
		)
		wantErr error
	}{
		{
			name: "success",
			setup: func(
				ctx context.Context,
				repo *mocks.MockcartRepository,
				productClient *mocks.MockproductClient,
				lomsClient *mocks.MocklomsClient,
			) {
				gomock.InOrder(
					productClient.EXPECT().
						GetProductInfo(ctx, uint32(1001)).
						Return(&port.ProductInfo{
							Name:  "milk",
							Price: 100,
						}, nil),

					lomsClient.EXPECT().
						GetStock(ctx, uint32(1001)).
						Return(uint64(10), nil),

					repo.EXPECT().
						AddItem(ctx, int64(10), entity.OrderItem{Sku: 1001, Count: 2}).
						Return(nil),
				)
			},
			wantErr: nil,
		},
		{
			name: "product not found",
			setup: func(
				ctx context.Context,
				repo *mocks.MockcartRepository,
				productClient *mocks.MockproductClient,
				lomsClient *mocks.MocklomsClient,
			) {
				productClient.EXPECT().
					GetProductInfo(ctx, uint32(1001)).
					Return(nil, port.ErrProductNotFound)
			},
			wantErr: xerror.ErrProductNotFound,
		},
		{
			name: "product client unexpected error",
			setup: func(
				ctx context.Context,
				repo *mocks.MockcartRepository,
				productClient *mocks.MockproductClient,
				lomsClient *mocks.MocklomsClient,
			) {
				productClient.EXPECT().
					GetProductInfo(ctx, uint32(1001)).
					Return(nil, errProductClient)
			},
			wantErr: errProductClient,
		},
		{
			name: "get stock error",
			setup: func(
				ctx context.Context,
				repo *mocks.MockcartRepository,
				productClient *mocks.MockproductClient,
				lomsClient *mocks.MocklomsClient,
			) {
				gomock.InOrder(
					productClient.EXPECT().
						GetProductInfo(ctx, uint32(1001)).
						Return(&port.ProductInfo{
							Name:  "milk",
							Price: 100,
						}, nil),

					lomsClient.EXPECT().
						GetStock(ctx, uint32(1001)).
						Return(uint64(0), errLomsClient),
				)
			},
			wantErr: errLomsClient,
		},
		{
			name: "insufficient stock",
			setup: func(
				ctx context.Context,
				repo *mocks.MockcartRepository,
				productClient *mocks.MockproductClient,
				lomsClient *mocks.MocklomsClient,
			) {
				gomock.InOrder(
					productClient.EXPECT().
						GetProductInfo(ctx, uint32(1001)).
						Return(&port.ProductInfo{
							Name:  "milk",
							Price: 100,
						}, nil),

					lomsClient.EXPECT().
						GetStock(ctx, uint32(1001)).
						Return(uint64(1), nil),
				)
			},
			wantErr: xerror.ErrInsufficientStock,
		},
		{
			name: "repository add item error",
			setup: func(
				ctx context.Context,
				repo *mocks.MockcartRepository,
				productClient *mocks.MockproductClient,
				lomsClient *mocks.MocklomsClient,
			) {
				gomock.InOrder(
					productClient.EXPECT().
						GetProductInfo(ctx, uint32(1001)).
						Return(&port.ProductInfo{
							Name:  "milk",
							Price: 100,
						}, nil),

					lomsClient.EXPECT().
						GetStock(ctx, uint32(1001)).
						Return(uint64(10), nil),

					repo.EXPECT().
						AddItem(ctx, int64(10), entity.OrderItem{Sku: 1001, Count: 2}).
						Return(errRepository),
				)
			},
			wantErr: errRepository,
		},
		{
			name: "stock equals requested count",
			setup: func(
				ctx context.Context,
				repo *mocks.MockcartRepository,
				productClient *mocks.MockproductClient,
				lomsClient *mocks.MocklomsClient,
			) {
				gomock.InOrder(
					productClient.EXPECT().
						GetProductInfo(ctx, uint32(1001)).
						Return(&port.ProductInfo{
							Name:  "milk",
							Price: 100,
						}, nil),

					lomsClient.EXPECT().
						GetStock(ctx, uint32(1001)).
						Return(uint64(2), nil),

					repo.EXPECT().
						AddItem(ctx, int64(10), entity.OrderItem{Sku: 1001, Count: 2}).
						Return(nil),
				)
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, service, repo, productClient, lomsClient := newTestService(t)

			tt.setup(ctx, repo, productClient, lomsClient)

			err := service.AddItem(ctx, 10, entity.OrderItem{Sku: 1001, Count: 2})

			if tt.wantErr != nil {
				requireErrorIs(t, err, tt.wantErr)
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestItemService_DeleteItem(t *testing.T) {
	t.Parallel()

	errRepository := errors.New("repository error")

	tests := []struct {
		name    string
		setup   func(ctx context.Context, repo *mocks.MockcartRepository)
		wantErr error
	}{
		{
			name: "success",
			setup: func(ctx context.Context, repo *mocks.MockcartRepository) {
				repo.EXPECT().
					DeleteItem(ctx, int64(10), uint32(1001)).
					Return(nil)
			},
			wantErr: nil,
		},
		{
			name: "repository delete item error",
			setup: func(ctx context.Context, repo *mocks.MockcartRepository) {
				repo.EXPECT().
					DeleteItem(ctx, int64(10), uint32(1001)).
					Return(errRepository)
			},
			wantErr: errRepository,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, service, repo, _, _ := newTestService(t)

			tt.setup(ctx, repo)

			err := service.DeleteItem(ctx, 10, 1001)

			if tt.wantErr != nil {
				requireErrorIs(t, err, tt.wantErr)
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
