package loms

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/controller/loms/mocks"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	xerror "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	lomspb "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/loms/v1"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestLomsServer_CreateOrder(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name     string
		req      *lomspb.CreateOrderRequest
		setup    func(*mocks.MocklomsService)
		wantCode codes.Code
		wantID   int64
	}

	tests := []testCase{
		{
			name: "success",
			req: &lomspb.CreateOrderRequest{
				UserId: 1,
				Items: []*lomspb.Item{
					{Sku: 1001, Count: 2},
				},
			},
			setup: func(s *mocks.MocklomsService) {
				s.EXPECT().
					CreateOrder(gomock.Any(), int64(1), []entity.Item{
						{Sku: 1001, Count: 2},
					}).
					Return(int64(10), nil)
			},
			wantCode: codes.OK,
			wantID:   10,
		},
		{
			name: "zero user id goes to service",
			req: &lomspb.CreateOrderRequest{
				UserId: 0,
			},
			setup: func(s *mocks.MocklomsService) {
				s.EXPECT().
					CreateOrder(gomock.Any(), int64(0), []entity.Item{}).
					Return(int64(0), errors.New("service error"))
			},
			wantCode: codes.Internal,
		},
		{
			name: "insufficient stock",
			req: &lomspb.CreateOrderRequest{
				UserId: 1,
				Items: []*lomspb.Item{
					{Sku: 1001, Count: 2},
				},
			},
			setup: func(s *mocks.MocklomsService) {
				s.EXPECT().
					CreateOrder(gomock.Any(), int64(1), []entity.Item{
						{Sku: 1001, Count: 2},
					}).
					Return(int64(0), xerror.ErrInsufficientStock)
			},
			wantCode: codes.FailedPrecondition,
		},
		{
			name: "internal error",
			req: &lomspb.CreateOrderRequest{
				UserId: 1,
				Items: []*lomspb.Item{
					{Sku: 1001, Count: 2},
				},
			},
			setup: func(s *mocks.MocklomsService) {
				s.EXPECT().
					CreateOrder(gomock.Any(), int64(1), []entity.Item{
						{Sku: 1001, Count: 2},
					}).
					Return(int64(0), errors.New("db error"))
			},
			wantCode: codes.Internal,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			service := mocks.NewMocklomsService(ctrl)
			test.setup(service)

			server := NewLomsServer(service, zap.NewNop())

			resp, err := server.CreateOrder(context.Background(), test.req)

			require.Equal(t, test.wantCode, status.Code(err))

			if test.wantCode == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Equal(t, test.wantID, resp.OrderId)
			} else {
				require.Error(t, err)
				require.Nil(t, resp)
			}
		})
	}
}

func TestLomsServer_GetOrder(t *testing.T) {
	t.Parallel()

	now := time.Now()

	type testCase struct {
		name     string
		req      *lomspb.GetOrderRequest
		setup    func(*mocks.MocklomsService)
		wantCode codes.Code
	}

	tests := []testCase{
		{
			name: "success",
			req:  &lomspb.GetOrderRequest{OrderId: 10},
			setup: func(s *mocks.MocklomsService) {
				s.EXPECT().
					GetOrder(gomock.Any(), int64(10)).
					Return(&entity.Order{
						UserID:    1,
						Status:    0,
						Items:     []entity.Item{{Sku: 1001, Count: 2}},
						CreatedAt: now,
						UpdatedAt: now,
					}, nil)
			},
			wantCode: codes.OK,
		},
		{
			name: "zero order id goes to service",
			req:  &lomspb.GetOrderRequest{OrderId: 0},
			setup: func(s *mocks.MocklomsService) {
				s.EXPECT().
					GetOrder(gomock.Any(), int64(0)).
					Return(nil, xerror.ErrOrderNotFound)
			},
			wantCode: codes.NotFound,
		},
		{
			name: "not found",
			req:  &lomspb.GetOrderRequest{OrderId: 10},
			setup: func(s *mocks.MocklomsService) {
				s.EXPECT().
					GetOrder(gomock.Any(), int64(10)).
					Return(nil, xerror.ErrOrderNotFound)
			},
			wantCode: codes.NotFound,
		},
		{
			name: "invalid status",
			req:  &lomspb.GetOrderRequest{OrderId: 10},
			setup: func(s *mocks.MocklomsService) {
				s.EXPECT().
					GetOrder(gomock.Any(), int64(10)).
					Return(nil, xerror.ErrInvalidStatus)
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "internal error",
			req:  &lomspb.GetOrderRequest{OrderId: 10},
			setup: func(s *mocks.MocklomsService) {
				s.EXPECT().
					GetOrder(gomock.Any(), int64(10)).
					Return(nil, errors.New("db error"))
			},
			wantCode: codes.Internal,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			service := mocks.NewMocklomsService(ctrl)
			test.setup(service)

			server := NewLomsServer(service, zap.NewNop())

			resp, err := server.GetOrder(context.Background(), test.req)

			require.Equal(t, test.wantCode, status.Code(err))

			if test.wantCode == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Equal(t, int64(1), resp.UserId)
				require.Len(t, resp.Items, 1)
				require.Equal(t, uint32(1001), resp.Items[0].Sku)
				require.Equal(t, uint32(2), resp.Items[0].Count)
			} else {
				require.Error(t, err)
				require.Nil(t, resp)
			}
		})
	}
}

func TestLomsServer_PayOrder(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name     string
		req      *lomspb.PayOrderRequest
		setup    func(*mocks.MocklomsService)
		wantCode codes.Code
	}

	tests := []testCase{
		{
			name: "success",
			req:  &lomspb.PayOrderRequest{OrderId: 10},
			setup: func(s *mocks.MocklomsService) {
				s.EXPECT().
					PayOrder(gomock.Any(), int64(10)).
					Return(nil)
			},
			wantCode: codes.OK,
		},
		{
			name: "zero order id goes to service",
			req:  &lomspb.PayOrderRequest{OrderId: 0},
			setup: func(s *mocks.MocklomsService) {
				s.EXPECT().
					PayOrder(gomock.Any(), int64(0)).
					Return(xerror.ErrOrderNotFound)
			},
			wantCode: codes.NotFound,
		},
		{
			name: "not found",
			req:  &lomspb.PayOrderRequest{OrderId: 10},
			setup: func(s *mocks.MocklomsService) {
				s.EXPECT().
					PayOrder(gomock.Any(), int64(10)).
					Return(xerror.ErrOrderNotFound)
			},
			wantCode: codes.NotFound,
		},
		{
			name: "invalid status",
			req:  &lomspb.PayOrderRequest{OrderId: 10},
			setup: func(s *mocks.MocklomsService) {
				s.EXPECT().
					PayOrder(gomock.Any(), int64(10)).
					Return(xerror.ErrInvalidStatus)
			},
			wantCode: codes.FailedPrecondition,
		},
		{
			name: "internal error",
			req:  &lomspb.PayOrderRequest{OrderId: 10},
			setup: func(s *mocks.MocklomsService) {
				s.EXPECT().
					PayOrder(gomock.Any(), int64(10)).
					Return(errors.New("db error"))
			},
			wantCode: codes.Internal,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			service := mocks.NewMocklomsService(ctrl)
			test.setup(service)

			server := NewLomsServer(service, zap.NewNop())

			resp, err := server.PayOrder(context.Background(), test.req)

			require.Equal(t, test.wantCode, status.Code(err))

			if test.wantCode == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, resp)
			} else {
				require.Error(t, err)
				require.Nil(t, resp)
			}
		})
	}
}

func TestLomsServer_CancelOrder(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name     string
		req      *lomspb.CancelOrderRequest
		setup    func(*mocks.MocklomsService)
		wantCode codes.Code
	}

	tests := []testCase{
		{
			name: "success",
			req:  &lomspb.CancelOrderRequest{OrderId: 10},
			setup: func(s *mocks.MocklomsService) {
				s.EXPECT().
					CancelOrder(gomock.Any(), int64(10)).
					Return(nil)
			},
			wantCode: codes.OK,
		},
		{
			name: "zero order id goes to service",
			req:  &lomspb.CancelOrderRequest{OrderId: 0},
			setup: func(s *mocks.MocklomsService) {
				s.EXPECT().
					CancelOrder(gomock.Any(), int64(0)).
					Return(xerror.ErrOrderNotFound)
			},
			wantCode: codes.NotFound,
		},
		{
			name: "not found",
			req:  &lomspb.CancelOrderRequest{OrderId: 10},
			setup: func(s *mocks.MocklomsService) {
				s.EXPECT().
					CancelOrder(gomock.Any(), int64(10)).
					Return(xerror.ErrOrderNotFound)
			},
			wantCode: codes.NotFound,
		},
		{
			name: "invalid status",
			req:  &lomspb.CancelOrderRequest{OrderId: 10},
			setup: func(s *mocks.MocklomsService) {
				s.EXPECT().
					CancelOrder(gomock.Any(), int64(10)).
					Return(xerror.ErrInvalidStatus)
			},
			wantCode: codes.FailedPrecondition,
		},
		{
			name: "internal error",
			req:  &lomspb.CancelOrderRequest{OrderId: 10},
			setup: func(s *mocks.MocklomsService) {
				s.EXPECT().
					CancelOrder(gomock.Any(), int64(10)).
					Return(errors.New("db error"))
			},
			wantCode: codes.Internal,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			service := mocks.NewMocklomsService(ctrl)
			test.setup(service)

			server := NewLomsServer(service, zap.NewNop())

			resp, err := server.CancelOrder(context.Background(), test.req)

			require.Equal(t, test.wantCode, status.Code(err))

			if test.wantCode == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, resp)
			} else {
				require.Error(t, err)
				require.Nil(t, resp)
			}
		})
	}
}

func TestLomsServer_GetOrder_AllStatuses(t *testing.T) {
	t.Parallel()

	now := time.Now()

	statuses := []entity.OrderStatus{
		entity.OrderStatus(0),
		entity.OrderStatus(1),
		entity.OrderStatus(2),
		entity.OrderStatus(3),
		entity.OrderStatus(4),
		entity.OrderStatus(999),
	}

	for _, orderStatus := range statuses {
		t.Run(string(rune(orderStatus)), func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			service := mocks.NewMocklomsService(ctrl)

			service.EXPECT().
				GetOrder(gomock.Any(), int64(10)).
				Return(&entity.Order{
					UserID: 1,
					Status: orderStatus,
					Items: []entity.Item{
						{Sku: 1001, Count: 2},
					},
					CreatedAt: now,
					UpdatedAt: now,
				}, nil)

			server := NewLomsServer(service, zap.NewNop())

			resp, err := server.GetOrder(context.Background(), &lomspb.GetOrderRequest{
				OrderId: 10,
			})

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Equal(t, int64(1), resp.UserId)
			require.Len(t, resp.Items, 1)
		})
	}
}
