package stocks

import (
	"context"
	"errors"
	"testing"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/controller/stocks/mocks"
	xerror "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	stockpb "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/stocks/v1"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestStocksServer_GetStock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		req      *stockpb.GetStockRequest
		mock     func(s *mocks.MockstocksService)
		want     *stockpb.GetStockResponse
		wantCode codes.Code
	}{
		{
			name: "success",
			req:  &stockpb.GetStockRequest{Sku: 1001},
			mock: func(s *mocks.MockstocksService) {
				s.EXPECT().
					GetStock(gomock.Any(), uint32(1001)).
					Return(uint64(42), nil)
			},
			want:     &stockpb.GetStockResponse{Count: 42},
			wantCode: codes.OK,
		},
		{
			name: "not found",
			req:  &stockpb.GetStockRequest{Sku: 1001},
			mock: func(s *mocks.MockstocksService) {
				s.EXPECT().
					GetStock(gomock.Any(), uint32(1001)).
					Return(uint64(0), xerror.ErrNotFoundProduct)
			},
			wantCode: codes.NotFound,
		},
		{
			name: "internal error",
			req:  &stockpb.GetStockRequest{Sku: 1001},
			mock: func(s *mocks.MockstocksService) {
				s.EXPECT().
					GetStock(gomock.Any(), uint32(1001)).
					Return(uint64(0), errors.New("db error"))
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			service := mocks.NewMockstocksService(ctrl)
			tt.mock(service)

			server := NewStocksServer(service)

			got, err := server.GetStock(context.Background(), tt.req)

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

func TestStocksServer_SetStock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		req      *stockpb.SetStockRequest
		mock     func(s *mocks.MockstocksService)
		wantCode codes.Code
	}{
		{
			name: "success",
			req:  &stockpb.SetStockRequest{Sku: 1001, Count: 42},
			mock: func(s *mocks.MockstocksService) {
				s.EXPECT().
					SetStock(gomock.Any(), uint32(1001), uint64(42)).
					Return(nil)
			},
			wantCode: codes.OK,
		},
		{
			name: "not found",
			req:  &stockpb.SetStockRequest{Sku: 1001, Count: 42},
			mock: func(s *mocks.MockstocksService) {
				s.EXPECT().
					SetStock(gomock.Any(), uint32(1001), uint64(42)).
					Return(xerror.ErrNotFoundProduct)
			},
			wantCode: codes.NotFound,
		},
		{
			name: "internal error",
			req:  &stockpb.SetStockRequest{Sku: 1001, Count: 42},
			mock: func(s *mocks.MockstocksService) {
				s.EXPECT().
					SetStock(gomock.Any(), uint32(1001), uint64(42)).
					Return(errors.New("db error"))
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			service := mocks.NewMockstocksService(ctrl)
			tt.mock(service)

			server := NewStocksServer(service)

			got, err := server.SetStock(context.Background(), tt.req)

			require.Equal(t, tt.wantCode, status.Code(err))

			if tt.wantCode == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, got)
			} else {
				require.Nil(t, got)
			}
		})
	}
}
