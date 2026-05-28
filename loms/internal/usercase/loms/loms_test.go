package loms

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	xerror "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/port"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/usercase/loms/mocks"
)

func setup(t *testing.T) (
	context.Context,
	*lomsService,
	*mocks.MockorderRepository,
	*mocks.MockstocksRepository,
	*mocks.MockoutboxRepository,
	*mocks.MocknotificationsClient,
	*mocks.Mocktransactor,
) {
	t.Helper()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	orders := mocks.NewMockorderRepository(ctrl)
	stocks := mocks.NewMockstocksRepository(ctrl)
	products := mocks.NewMockproductRepository(ctrl)
	outbox := mocks.NewMockoutboxRepository(ctrl)
	notifications := mocks.NewMocknotificationsClient(ctrl)
	tx := mocks.NewMocktransactor(ctrl)

	svc := NewLomsService(orders, stocks, products, outbox, notifications, tx)

	return context.Background(), svc, orders, stocks, outbox, notifications, tx
}

func mockTx(ctx context.Context, tx *mocks.Mocktransactor) {
	tx.EXPECT().
		WithTx(ctx, gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})
}

func assertErr(t *testing.T, got, want error) {
	t.Helper()

	if want == nil && got != nil {
		t.Fatalf("unexpected error: %v", got)
	}

	if want != nil && !errors.Is(got, want) {
		t.Fatalf("got error: %v, want: %v", got, want)
	}
}

type outboxMatcher struct {
	orderID int64
	UserID  int64
	status  port.OrderStatus
}

func (m outboxMatcher) Matches(x any) bool {
	msg, ok := x.(*entity.OutboxMessage)
	if !ok || msg.Kind != entity.KindNotification {
		return false
	}

	var body port.OrderStatusChangedNotification
	if err := json.Unmarshal(msg.Data, &body); err != nil {
		return false
	}

	return body.OrderID == m.orderID &&
		body.UserID == m.UserID &&
		body.Status == m.status
}

func (m outboxMatcher) String() string {
	return "outbox order status changed notification"
}

func TestLomsService_CreateOrder(t *testing.T) {
	t.Parallel()

	errAny := errors.New("any error")

	tests := []struct {
		name    string
		setup   func(context.Context, *mocks.MockorderRepository, *mocks.MockstocksRepository, *mocks.MockoutboxRepository, *mocks.Mocktransactor)
		wantID  int64
		wantErr error
	}{
		{
			name: "success",
			setup: func(ctx context.Context, orders *mocks.MockorderRepository, stocks *mocks.MockstocksRepository, outbox *mocks.MockoutboxRepository, tx *mocks.Mocktransactor) {
				mockTx(ctx, tx)

				items := []entity.Item{{Sku: 1001, Count: 2}}

				gomock.InOrder(
					stocks.EXPECT().ReserveStock(ctx, uint32(1001), uint64(2)).Return(nil),
					orders.EXPECT().CreateOrder(ctx, &entity.Order{
						UserID: 10,
						Items:  items,
						Status: entity.OrderStatusAwaitingPayment,
					}).Return(int64(777), nil),
					outbox.EXPECT().SendOutboxMessage(ctx, outboxMatcher{
						orderID: 777,
						UserID:  10,
						status:  port.OrderStatusAwaitingPayment,
					}).Return(nil),
				)
			},
			wantID: 777,
		},
		{
			name: "reserve error",
			setup: func(ctx context.Context, orders *mocks.MockorderRepository, stocks *mocks.MockstocksRepository, outbox *mocks.MockoutboxRepository, tx *mocks.Mocktransactor) {
				mockTx(ctx, tx)

				stocks.EXPECT().ReserveStock(ctx, uint32(1001), uint64(2)).Return(errAny)
			},
			wantErr: errAny,
		},
		{
			name: "create order error",
			setup: func(ctx context.Context, orders *mocks.MockorderRepository, stocks *mocks.MockstocksRepository, outbox *mocks.MockoutboxRepository, tx *mocks.Mocktransactor) {
				mockTx(ctx, tx)

				items := []entity.Item{{Sku: 1001, Count: 2}}

				stocks.EXPECT().ReserveStock(ctx, uint32(1001), uint64(2)).Return(nil)
				orders.EXPECT().CreateOrder(ctx, &entity.Order{
					UserID: 10,
					Items:  items,
					Status: entity.OrderStatusAwaitingPayment,
				}).Return(int64(0), errAny)
			},
			wantErr: errAny,
		},
		{
			name: "outbox error",
			setup: func(ctx context.Context, orders *mocks.MockorderRepository, stocks *mocks.MockstocksRepository, outbox *mocks.MockoutboxRepository, tx *mocks.Mocktransactor) {
				mockTx(ctx, tx)

				items := []entity.Item{{Sku: 1001, Count: 2}}

				stocks.EXPECT().ReserveStock(ctx, uint32(1001), uint64(2)).Return(nil)
				orders.EXPECT().CreateOrder(ctx, &entity.Order{
					UserID: 10,
					Items:  items,
					Status: entity.OrderStatusAwaitingPayment,
				}).Return(int64(777), nil)
				outbox.EXPECT().SendOutboxMessage(ctx, outboxMatcher{
					orderID: 777,
					UserID:  10,
					status:  port.OrderStatusAwaitingPayment,
				}).Return(errAny)
			},
			wantErr: errAny,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, svc, orders, stocks, outbox, _, tx := setup(t)
			tt.setup(ctx, orders, stocks, outbox, tx)

			gotID, err := svc.CreateOrder(ctx, 10, []entity.Item{{Sku: 1001, Count: 2}})

			if gotID != tt.wantID {
				t.Fatalf("got id: %d, want: %d", gotID, tt.wantID)
			}

			assertErr(t, err, tt.wantErr)
		})
	}
}

func TestLomsService_GetOrder(t *testing.T) {
	t.Parallel()

	ctx, svc, orders, _, _, _, _ := setup(t)

	want := &entity.Order{
		ID:     777,
		UserID: 10,
		Status: entity.OrderStatusAwaitingPayment,
	}

	orders.EXPECT().GetOrder(ctx, int64(777)).Return(want, nil)

	got, err := svc.GetOrder(ctx, 777)

	assertErr(t, err, nil)

	if got != want {
		t.Fatalf("got: %#v, want: %#v", got, want)
	}
}

func TestLomsService_PayOrder(t *testing.T) {
	t.Parallel()

	errAny := errors.New("any error")

	tests := []struct {
		name    string
		status  entity.OrderStatus
		setup   func(context.Context, *mocks.MockorderRepository, *mocks.MockoutboxRepository, *mocks.Mocktransactor)
		wantErr error
	}{
		{
			name:   "success",
			status: entity.OrderStatusAwaitingPayment,
			setup: func(ctx context.Context, orders *mocks.MockorderRepository, outbox *mocks.MockoutboxRepository, tx *mocks.Mocktransactor) {
				mockTx(ctx, tx)

				order := &entity.Order{
					ID:     777,
					UserID: 10,
					Status: entity.OrderStatusAwaitingPayment,
				}

				gomock.InOrder(
					orders.EXPECT().GetOrderForUpdate(ctx, int64(777)).Return(order, nil),
					orders.EXPECT().UpdateStatusOrder(ctx, int64(777), entity.OrderStatusPaid).Return(nil),
					outbox.EXPECT().SendOutboxMessage(ctx, outboxMatcher{
						orderID: 777,
						UserID:  10,
						status:  port.OrderStatusPaid,
					}).Return(nil),
				)
			},
		},
		{
			name: "get order error",
			setup: func(ctx context.Context, orders *mocks.MockorderRepository, outbox *mocks.MockoutboxRepository, tx *mocks.Mocktransactor) {
				mockTx(ctx, tx)

				orders.EXPECT().GetOrderForUpdate(ctx, int64(777)).Return(nil, errAny)
			},
			wantErr: errAny,
		},
		{
			name:   "invalid status",
			status: entity.OrderStatusPaid,
			setup: func(ctx context.Context, orders *mocks.MockorderRepository, outbox *mocks.MockoutboxRepository, tx *mocks.Mocktransactor) {
				mockTx(ctx, tx)

				orders.EXPECT().GetOrderForUpdate(ctx, int64(777)).Return(&entity.Order{
					ID:     777,
					UserID: 10,
					Status: entity.OrderStatusPaid,
				}, nil)
			},
			wantErr: xerror.ErrInvalidStatus,
		},
		{
			name:   "update error",
			status: entity.OrderStatusAwaitingPayment,
			setup: func(ctx context.Context, orders *mocks.MockorderRepository, outbox *mocks.MockoutboxRepository, tx *mocks.Mocktransactor) {
				mockTx(ctx, tx)

				orders.EXPECT().GetOrderForUpdate(ctx, int64(777)).Return(&entity.Order{
					ID:     777,
					UserID: 10,
					Status: entity.OrderStatusAwaitingPayment,
				}, nil)
				orders.EXPECT().UpdateStatusOrder(ctx, int64(777), entity.OrderStatusPaid).Return(errAny)
			},
			wantErr: errAny,
		},
		{
			name:   "outbox error",
			status: entity.OrderStatusAwaitingPayment,
			setup: func(ctx context.Context, orders *mocks.MockorderRepository, outbox *mocks.MockoutboxRepository, tx *mocks.Mocktransactor) {
				mockTx(ctx, tx)

				order := &entity.Order{
					ID:     777,
					UserID: 10,
					Status: entity.OrderStatusAwaitingPayment,
				}

				orders.EXPECT().GetOrderForUpdate(ctx, int64(777)).Return(order, nil)
				orders.EXPECT().UpdateStatusOrder(ctx, int64(777), entity.OrderStatusPaid).Return(nil)
				outbox.EXPECT().SendOutboxMessage(ctx, outboxMatcher{
					orderID: 777,
					UserID:  10,
					status:  port.OrderStatusPaid,
				}).Return(errAny)
			},
			wantErr: errAny,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, svc, orders, _, outbox, _, tx := setup(t)
			tt.setup(ctx, orders, outbox, tx)

			err := svc.PayOrder(ctx, 777)

			assertErr(t, err, tt.wantErr)
		})
	}
}

func TestLomsService_CancelOrder(t *testing.T) {
	t.Parallel()

	errAny := errors.New("any error")

	tests := []struct {
		name    string
		setup   func(context.Context, *mocks.MockorderRepository, *mocks.MockstocksRepository, *mocks.MockoutboxRepository, *mocks.Mocktransactor)
		wantErr error
	}{
		{
			name: "success",
			setup: func(ctx context.Context, orders *mocks.MockorderRepository, stocks *mocks.MockstocksRepository, outbox *mocks.MockoutboxRepository, tx *mocks.Mocktransactor) {
				mockTx(ctx, tx)

				order := &entity.Order{
					ID:     777,
					UserID: 10,
					Status: entity.OrderStatusAwaitingPayment,
					Items:  []entity.Item{{Sku: 1001, Count: 2}},
				}

				gomock.InOrder(
					orders.EXPECT().GetOrderForUpdate(ctx, int64(777)).Return(order, nil),
					stocks.EXPECT().ReleaseStock(ctx, uint32(1001), uint64(2)).Return(nil),
					orders.EXPECT().UpdateStatusOrder(ctx, int64(777), entity.OrderStatusCancelled).Return(nil),
					outbox.EXPECT().SendOutboxMessage(ctx, outboxMatcher{
						orderID: 777,
						UserID:  10,
						status:  port.OrderStatusCancelled,
					}).Return(nil),
				)
			},
		},
		{
			name: "invalid status",
			setup: func(ctx context.Context, orders *mocks.MockorderRepository, stocks *mocks.MockstocksRepository, outbox *mocks.MockoutboxRepository, tx *mocks.Mocktransactor) {
				mockTx(ctx, tx)

				orders.EXPECT().GetOrderForUpdate(ctx, int64(777)).Return(&entity.Order{
					ID:     777,
					UserID: 10,
					Status: entity.OrderStatusPaid,
				}, nil)
			},
			wantErr: xerror.ErrInvalidStatus,
		},
		{
			name: "release error",
			setup: func(ctx context.Context, orders *mocks.MockorderRepository, stocks *mocks.MockstocksRepository, outbox *mocks.MockoutboxRepository, tx *mocks.Mocktransactor) {
				mockTx(ctx, tx)

				orders.EXPECT().GetOrderForUpdate(ctx, int64(777)).Return(&entity.Order{
					ID:     777,
					UserID: 10,
					Status: entity.OrderStatusAwaitingPayment,
					Items:  []entity.Item{{Sku: 1001, Count: 2}},
				}, nil)
				stocks.EXPECT().ReleaseStock(ctx, uint32(1001), uint64(2)).Return(errAny)
			},
			wantErr: errAny,
		},
		{
			name: "update error",
			setup: func(ctx context.Context, orders *mocks.MockorderRepository, stocks *mocks.MockstocksRepository, outbox *mocks.MockoutboxRepository, tx *mocks.Mocktransactor) {
				mockTx(ctx, tx)

				orders.EXPECT().GetOrderForUpdate(ctx, int64(777)).Return(&entity.Order{
					ID:     777,
					UserID: 10,
					Status: entity.OrderStatusAwaitingPayment,
					Items:  []entity.Item{{Sku: 1001, Count: 2}},
				}, nil)
				stocks.EXPECT().ReleaseStock(ctx, uint32(1001), uint64(2)).Return(nil)
				orders.EXPECT().UpdateStatusOrder(ctx, int64(777), entity.OrderStatusCancelled).Return(errAny)
			},
			wantErr: errAny,
		},
		{
			name: "outbox error",
			setup: func(ctx context.Context, orders *mocks.MockorderRepository, stocks *mocks.MockstocksRepository, outbox *mocks.MockoutboxRepository, tx *mocks.Mocktransactor) {
				mockTx(ctx, tx)

				order := &entity.Order{
					ID:     777,
					UserID: 10,
					Status: entity.OrderStatusAwaitingPayment,
					Items:  []entity.Item{{Sku: 1001, Count: 2}},
				}

				orders.EXPECT().GetOrderForUpdate(ctx, int64(777)).Return(order, nil)
				stocks.EXPECT().ReleaseStock(ctx, uint32(1001), uint64(2)).Return(nil)
				orders.EXPECT().UpdateStatusOrder(ctx, int64(777), entity.OrderStatusCancelled).Return(nil)
				outbox.EXPECT().SendOutboxMessage(ctx, outboxMatcher{
					orderID: 777,
					UserID:  10,
					status:  port.OrderStatusCancelled,
				}).Return(errAny)
			},
			wantErr: errAny,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, svc, orders, stocks, outbox, _, tx := setup(t)
			tt.setup(ctx, orders, stocks, outbox, tx)

			err := svc.CancelOrder(ctx, 777)

			assertErr(t, err, tt.wantErr)
		})
	}
}

func TestLomsService_OrderStatusChangedNotificationKindHandler(t *testing.T) {
	t.Parallel()

	errAny := errors.New("any error")

	tests := []struct {
		name    string
		body    []byte
		setup   func(context.Context, *mocks.MocknotificationsClient)
		wantErr bool
	}{
		{
			name: "success",
			body: mustJSON(t, port.OrderStatusChangedNotification{
				OrderID: 777,
				UserID:  10,
				Status:  port.OrderStatusPaid,
			}),
			setup: func(ctx context.Context, n *mocks.MocknotificationsClient) {
				n.EXPECT().
					SendOrderStatusChangedNotification(ctx, int64(10), int64(777), port.OrderStatusPaid).
					Return(nil)
			},
		},
		{
			name: "bad json",
			body: []byte(`{bad json`),
			setup: func(ctx context.Context, n *mocks.MocknotificationsClient) {
			},
			wantErr: true,
		},
		{
			name: "client error",
			body: mustJSON(t, port.OrderStatusChangedNotification{
				OrderID: 777,
				UserID:  10,
				Status:  port.OrderStatusCancelled,
			}),
			setup: func(ctx context.Context, n *mocks.MocknotificationsClient) {
				n.EXPECT().
					SendOrderStatusChangedNotification(ctx, int64(10), int64(777), port.OrderStatusCancelled).
					Return(errAny)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, svc, _, _, _, notifications, _ := setup(t)
			tt.setup(ctx, notifications)

			err := svc.OrderStatusChangedNotificationKindHandler(ctx, tt.body)

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func mustJSON(t *testing.T, body port.OrderStatusChangedNotification) []byte {
	t.Helper()

	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}

	return data
}
