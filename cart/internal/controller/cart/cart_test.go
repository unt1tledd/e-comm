package cart

import (
	"context"
	"errors"
	"testing"

	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/controller/cart/mocks"
	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/entity"
	cartpb "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/cart/api/cart/v1"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type listCartStream struct {
	ctx  context.Context
	sent []*cartpb.ListCartResponse
	err  error
}

func (s *listCartStream) Send(resp *cartpb.ListCartResponse) error {
	if s.err != nil {
		return s.err
	}
	s.sent = append(s.sent, resp)
	return nil
}

func (s *listCartStream) Context() context.Context     { return s.ctx }
func (s *listCartStream) SetHeader(metadata.MD) error  { return nil }
func (s *listCartStream) SendHeader(metadata.MD) error { return nil }
func (s *listCartStream) SetTrailer(metadata.MD)       {}
func (s *listCartStream) SendMsg(any) error            { return nil }
func (s *listCartStream) RecvMsg(any) error            { return nil }

func newServer(t *testing.T) (*cartServer, *mocks.MockItemService, *mocks.MockService) {
	t.Helper()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	itemService := mocks.NewMockItemService(ctrl)
	cartService := mocks.NewMockService(ctrl)

	return NewCartServer(itemService, cartService, zap.NewNop()), itemService, cartService
}

func TestCartServer_AddItem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		req      *cartpb.AddItemRequest
		mock     func(*mocks.MockItemService)
		wantCode codes.Code
	}{
		{
			name: "invalid request",
			req:  &cartpb.AddItemRequest{},
			mock: func(s *mocks.MockItemService) {
				s.EXPECT().
					AddItem(gomock.Any(), int64(0), entity.OrderItem{Sku: 0, Count: 0}).
					Return(errors.New("invalid request"))
			},
			wantCode: codes.Internal,
		},
		{
			name: "success",
			req:  &cartpb.AddItemRequest{UserId: 1, Sku: 100, Count: 2},
			mock: func(s *mocks.MockItemService) {
				s.EXPECT().
					AddItem(gomock.Any(), int64(1), entity.OrderItem{Sku: 100, Count: 2}).
					Return(nil)
			},
			wantCode: codes.OK,
		},
		{
			name: "internal error",
			req:  &cartpb.AddItemRequest{UserId: 1, Sku: 100, Count: 2},
			mock: func(s *mocks.MockItemService) {
				s.EXPECT().
					AddItem(gomock.Any(), int64(1), entity.OrderItem{Sku: 100, Count: 2}).
					Return(errors.New("db error"))
			},
			wantCode: codes.Internal,
		},
	}

	for i := range tests {
		tc := tests[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s, itemService, _ := newServer(t)
			tc.mock(itemService)

			_, err := s.AddItem(context.Background(), tc.req)
			require.Equal(t, tc.wantCode, status.Code(err))
		})
	}
}

func TestCartServer_DeleteItem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		req      *cartpb.DeleteItemRequest
		mock     func(*mocks.MockItemService)
		wantCode codes.Code
	}{
		{
			name: "invalid request",
			req:  &cartpb.DeleteItemRequest{},
			mock: func(s *mocks.MockItemService) {
				s.EXPECT().
					DeleteItem(gomock.Any(), int64(0), uint32(0)).
					Return(errors.New("invalid request"))
			},
			wantCode: codes.Internal,
		},
		{
			name: "success",
			req:  &cartpb.DeleteItemRequest{UserId: 1, Sku: 100},
			mock: func(s *mocks.MockItemService) {
				s.EXPECT().
					DeleteItem(gomock.Any(), int64(1), uint32(100)).
					Return(nil)
			},
			wantCode: codes.OK,
		},
		{
			name: "internal error",
			req:  &cartpb.DeleteItemRequest{UserId: 1, Sku: 100},
			mock: func(s *mocks.MockItemService) {
				s.EXPECT().
					DeleteItem(gomock.Any(), int64(1), uint32(100)).
					Return(errors.New("db error"))
			},
			wantCode: codes.Internal,
		},
	}

	for i := range tests {
		tc := tests[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s, itemService, _ := newServer(t)
			tc.mock(itemService)

			_, err := s.DeleteItem(context.Background(), tc.req)
			require.Equal(t, tc.wantCode, status.Code(err))
		})
	}
}

func TestCartServer_ClearCart(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		req      *cartpb.ClearCartRequest
		mock     func(*mocks.MockService)
		wantCode codes.Code
	}{
		{
			name: "invalid request",
			req:  &cartpb.ClearCartRequest{},
			mock: func(s *mocks.MockService) {
				s.EXPECT().
					ClearCart(gomock.Any(), int64(0)).
					Return(errors.New("invalid request"))
			},
			wantCode: codes.Internal,
		},
		{
			name: "success",
			req:  &cartpb.ClearCartRequest{UserId: 1},
			mock: func(s *mocks.MockService) {
				s.EXPECT().
					ClearCart(gomock.Any(), int64(1)).
					Return(nil)
			},
			wantCode: codes.OK,
		},
		{
			name: "internal error",
			req:  &cartpb.ClearCartRequest{UserId: 1},
			mock: func(s *mocks.MockService) {
				s.EXPECT().
					ClearCart(gomock.Any(), int64(1)).
					Return(errors.New("db error"))
			},
			wantCode: codes.Internal,
		},
	}

	for i := range tests {
		tc := tests[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s, _, cartService := newServer(t)
			tc.mock(cartService)

			_, err := s.ClearCart(context.Background(), tc.req)
			require.Equal(t, tc.wantCode, status.Code(err))
		})
	}
}

func TestCartServer_CheckoutCart(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		req      *cartpb.CheckoutCartRequest
		mock     func(*mocks.MockService)
		want     *cartpb.CheckoutCartResponse
		wantCode codes.Code
	}{
		{
			name: "invalid request",
			req:  &cartpb.CheckoutCartRequest{},
			mock: func(s *mocks.MockService) {
				s.EXPECT().
					CheckoutCart(gomock.Any(), int64(0)).
					Return(int64(0), errors.New("invalid request"))
			},
			wantCode: codes.Internal,
		},
		{
			name: "success",
			req:  &cartpb.CheckoutCartRequest{UserId: 1},
			mock: func(s *mocks.MockService) {
				s.EXPECT().
					CheckoutCart(gomock.Any(), int64(1)).
					Return(int64(10), nil)
			},
			want:     &cartpb.CheckoutCartResponse{OrderId: 10},
			wantCode: codes.OK,
		},
		{
			name: "internal error",
			req:  &cartpb.CheckoutCartRequest{UserId: 1},
			mock: func(s *mocks.MockService) {
				s.EXPECT().
					CheckoutCart(gomock.Any(), int64(1)).
					Return(int64(0), errors.New("db error"))
			},
			wantCode: codes.Internal,
		},
	}

	for i := range tests {
		tc := tests[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s, _, cartService := newServer(t)
			tc.mock(cartService)

			got, err := s.CheckoutCart(context.Background(), tc.req)
			require.Equal(t, tc.wantCode, status.Code(err))
			require.Equal(t, tc.want, got)
		})
	}
}

func TestCartServer_ListCart(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		req       *cartpb.ListCartRequest
		mock      func(*mocks.MockService)
		streamErr error
		wantSent  []*cartpb.ListCartResponse
		wantCode  codes.Code
	}{
		{
			name: "invalid request",
			req:  &cartpb.ListCartRequest{},
			mock: func(s *mocks.MockService) {
				s.EXPECT().
					ListCart(gomock.Any(), int64(0)).
					Return(entity.CartInfo{}, errors.New("invalid request"))
			},
			wantCode: codes.Internal,
		},
		{
			name: "success",
			req:  &cartpb.ListCartRequest{UserId: 1},
			mock: func(s *mocks.MockService) {
				s.EXPECT().
					ListCart(gomock.Any(), int64(1)).
					Return(
						entity.CartInfo{
							Items: []entity.Product{{
								Sku:   100,
								Count: 2,
								Name:  "phone",
								Price: 500,
							}},
							TotalPrice: 1000,
						},
						nil,
					)
			},
			wantSent: []*cartpb.ListCartResponse{
				{
					Items: []*cartpb.Item{
						{
							Sku:   100,
							Count: 2,
							Name:  "phone",
							Price: 500,
						},
					},
					TotalPrice: 1000,
				},
			},
			wantCode: codes.OK,
		},
		{
			name: "empty cart",
			req:  &cartpb.ListCartRequest{UserId: 1},
			mock: func(s *mocks.MockService) {
				s.EXPECT().
					ListCart(gomock.Any(), int64(1)).
					Return(entity.CartInfo{}, nil)
			},
			wantCode: codes.OK,
		},
		{
			name: "service error",
			req:  &cartpb.ListCartRequest{UserId: 1},
			mock: func(s *mocks.MockService) {
				s.EXPECT().
					ListCart(gomock.Any(), int64(1)).
					Return(entity.CartInfo{}, errors.New("db error"))
			},
			wantCode: codes.Internal,
		},
		{
			name: "send error",
			req:  &cartpb.ListCartRequest{UserId: 1},
			mock: func(s *mocks.MockService) {
				s.EXPECT().
					ListCart(gomock.Any(), int64(1)).
					Return(
						entity.CartInfo{
							Items: []entity.Product{{
								Sku:   100,
								Count: 1,
								Name:  "phone",
								Price: 500,
							}},
							TotalPrice: 500,
						},
						nil,
					)
			},
			streamErr: errors.New("send error"),
			wantCode:  codes.Internal,
		},
	}

	for i := range tests {
		tc := tests[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s, _, cartService := newServer(t)
			tc.mock(cartService)

			stream := &listCartStream{
				ctx: context.Background(),
				err: tc.streamErr,
			}

			err := s.ListCart(tc.req, stream)

			require.Equal(t, tc.wantCode, status.Code(err))
			require.Equal(t, tc.wantSent, stream.sent)
		})
	}
}
