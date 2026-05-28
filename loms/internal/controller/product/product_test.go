package product

import (
	"context"
	"errors"
	"testing"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/controller/product/mocks"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	xerror "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	productpb "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/product/v1"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestProductServer_GetProduct(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		req      *productpb.GetProductRequest
		mock     func(s *mocks.MockproductService)
		want     *productpb.GetProductResponse
		wantCode codes.Code
	}{
		{
			name: "success",
			req:  &productpb.GetProductRequest{Sku: 1001},
			mock: func(s *mocks.MockproductService) {
				s.EXPECT().
					GetProduct(gomock.Any(), uint32(1001)).
					Return(&entity.ProductInfo{
						Name:  "test",
						Price: 100,
					}, nil)
			},
			want: &productpb.GetProductResponse{
				Name:  "test",
				Price: 100,
			},
			wantCode: codes.OK,
		},
		{
			name: "not found",
			req:  &productpb.GetProductRequest{Sku: 1001},
			mock: func(s *mocks.MockproductService) {
				s.EXPECT().
					GetProduct(gomock.Any(), uint32(1001)).
					Return(nil, xerror.ErrNotFoundProduct)
			},
			wantCode: codes.NotFound,
		},
		{
			name: "internal error",
			req:  &productpb.GetProductRequest{Sku: 1001},
			mock: func(s *mocks.MockproductService) {
				s.EXPECT().
					GetProduct(gomock.Any(), uint32(1001)).
					Return(nil, errors.New("db error"))
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			service := mocks.NewMockproductService(ctrl)
			tt.mock(service)

			server := NewProductServer(service)

			got, err := server.GetProduct(context.Background(), tt.req)

			require.Equal(t, tt.wantCode, status.Code(err))

			if tt.wantCode == codes.OK {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			} else {
				require.Nil(t, got)
			}
		})
	}
}

func TestProductServer_CreateProduct(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		req      *productpb.CreateProductRequest
		mock     func(s *mocks.MockproductService)
		want     *productpb.CreateProductResponse
		wantCode codes.Code
	}{
		{
			name: "success",
			req: &productpb.CreateProductRequest{
				Name:  "test",
				Price: 100,
			},
			mock: func(s *mocks.MockproductService) {
				s.EXPECT().
					CreateProduct(gomock.Any(), "test", uint32(100)).
					Return(uint32(123), nil)
			},
			want: &productpb.CreateProductResponse{
				Sku: 123,
			},
			wantCode: codes.OK,
		},
		{
			name: "internal error",
			req: &productpb.CreateProductRequest{
				Name:  "test",
				Price: 100,
			},
			mock: func(s *mocks.MockproductService) {
				s.EXPECT().
					CreateProduct(gomock.Any(), "test", uint32(100)).
					Return(uint32(0), errors.New("db error"))
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			service := mocks.NewMockproductService(ctrl)
			tt.mock(service)

			server := NewProductServer(service)

			got, err := server.CreateProduct(context.Background(), tt.req)

			require.Equal(t, tt.wantCode, status.Code(err))

			if tt.wantCode == codes.OK {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			} else {
				require.Nil(t, got)
			}
		})
	}
}
