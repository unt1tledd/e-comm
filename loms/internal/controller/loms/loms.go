package loms

import (
	"context"
	"errors"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/controller/converter"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	xerror "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	lomspb "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/loms/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

//go:generate mockgen -source=loms.go -destination=mocks/loms_mocks.go -package=mocks

type (
	lomsService interface {
		CreateOrder(ctx context.Context, userID int64, items []entity.Item) (orderID int64, err error)
		GetOrder(ctx context.Context, orderID int64) (order *entity.Order, err error)
		PayOrder(ctx context.Context, orderID int64) error
		CancelOrder(ctx context.Context, orderID int64) error
	}
)

type lomsServer struct {
	lomsService lomsService
	logger      *zap.Logger
}

func NewLomsServer(lomsService lomsService, logger *zap.Logger) *lomsServer {
	return &lomsServer{
		lomsService: lomsService,
		logger:      logger,
	}
}

func (l *lomsServer) CreateOrder(ctx context.Context, req *lomspb.CreateOrderRequest) (*lomspb.CreateOrderResponse, error) {
	if err := req.Validate(); err != nil {
		l.logger.Warn("create order validation failed", zap.Error(err))
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	orderID, err := l.lomsService.CreateOrder(ctx, req.GetUserId(), protoToEntity(req.GetItems()))
	if err != nil {
		switch {
		case errors.Is(err, xerror.ErrInsufficientStock):
			l.logger.Warn("create order failed: insufficient stock",
				zap.Int64("user_id", req.GetUserId()),
				zap.Error(err),
			)
			return nil, status.Error(codes.FailedPrecondition, "insufficient stock")
		default:
			l.logger.Error("create order failed",
				zap.Int64("user_id", req.GetUserId()),
				zap.Error(err),
			)
			return nil, status.Error(codes.Internal, "create order failed")
		}
	}

	l.logger.Info("order created",
		zap.Int64("user_id", req.GetUserId()),
		zap.Int64("order_id", orderID),
	)

	return &lomspb.CreateOrderResponse{OrderId: orderID}, nil
}

func (l *lomsServer) GetOrder(ctx context.Context, req *lomspb.GetOrderRequest) (*lomspb.GetOrderResponse, error) {
	if err := req.Validate(); err != nil {
		l.logger.Warn("get order validation failed", zap.Error(err))
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	order, err := l.lomsService.GetOrder(ctx, req.GetOrderId())
	if err != nil {
		switch {
		case errors.Is(err, xerror.ErrOrderNotFound):
			l.logger.Warn("order not found",
				zap.Int64("order_id", req.GetOrderId()),
				zap.Error(err),
			)
			return nil, status.Error(codes.NotFound, "order not found")
		case errors.Is(err, xerror.ErrInvalidStatus):
			l.logger.Warn("get order invalid status",
				zap.Int64("order_id", req.GetOrderId()),
				zap.Error(err),
			)
			return nil, status.Error(codes.InvalidArgument, "invalid status")
		default:
			l.logger.Error("get order failed",
				zap.Int64("order_id", req.GetOrderId()),
				zap.Error(err),
			)
			return nil, status.Error(codes.Internal, "get order failed")
		}
	}

	l.logger.Info("order fetched",
		zap.Int64("order_id", req.GetOrderId()),
		zap.Int64("user_id", order.UserID),
		zap.Int("status", int(order.Status)),
	)

	return &lomspb.GetOrderResponse{
		Status:    converter.FromOrderStatus(order.Status),
		UserId:    order.UserID,
		Items:     entityToProto(order.Items),
		CreatedAt: timestamppb.New(order.CreatedAt),
		UpdatedAt: timestamppb.New(order.UpdatedAt),
	}, nil
}

func (l *lomsServer) PayOrder(ctx context.Context, req *lomspb.PayOrderRequest) (*emptypb.Empty, error) {
	if err := req.Validate(); err != nil {
		l.logger.Warn("pay order validation failed", zap.Error(err))
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	err := l.lomsService.PayOrder(ctx, req.GetOrderId())
	if err != nil {
		switch {
		case errors.Is(err, xerror.ErrOrderNotFound):
			l.logger.Warn("pay order not found",
				zap.Int64("order_id", req.GetOrderId()),
				zap.Error(err),
			)
			return nil, status.Errorf(codes.NotFound, "%v", err)
		case errors.Is(err, xerror.ErrInvalidStatus):
			l.logger.Warn("pay order invalid status",
				zap.Int64("order_id", req.GetOrderId()),
				zap.Error(err),
			)
			return nil, status.Errorf(codes.FailedPrecondition, "%v", err)
		default:
			l.logger.Error("pay order failed",
				zap.Int64("order_id", req.GetOrderId()),
				zap.Error(err),
			)
			return nil, status.Error(codes.Internal, "pay order failed")
		}
	}

	l.logger.Info("order paid", zap.Int64("order_id", req.GetOrderId()))

	return &emptypb.Empty{}, nil
}

func (l *lomsServer) CancelOrder(ctx context.Context, req *lomspb.CancelOrderRequest) (*emptypb.Empty, error) {
	if err := req.Validate(); err != nil {
		l.logger.Warn("cancel order validation failed", zap.Error(err))
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	err := l.lomsService.CancelOrder(ctx, req.GetOrderId())
	if err != nil {
		switch {
		case errors.Is(err, xerror.ErrOrderNotFound):
			l.logger.Warn("cancel order not found",
				zap.Int64("order_id", req.GetOrderId()),
				zap.Error(err),
			)
			return nil, status.Errorf(codes.NotFound, "%v", err)
		case errors.Is(err, xerror.ErrInvalidStatus):
			l.logger.Warn("cancel order invalid status",
				zap.Int64("order_id", req.GetOrderId()),
				zap.Error(err),
			)
			return nil, status.Errorf(codes.FailedPrecondition, "%v", err)
		default:
			l.logger.Error("cancel order failed",
				zap.Int64("order_id", req.GetOrderId()),
				zap.Error(err),
			)
			return nil, status.Error(codes.Internal, "cancel order failed")
		}
	}

	l.logger.Info("order cancelled", zap.Int64("order_id", req.GetOrderId()))

	return &emptypb.Empty{}, nil
}

func entityToProto(items []entity.Item) []*lomspb.Item {
	itemspb := make([]*lomspb.Item, len(items))

	for i := range items {
		itemspb[i] = &lomspb.Item{
			Sku:   items[i].Sku,
			Count: items[i].Count,
		}
	}

	return itemspb
}

func protoToEntity(itemspb []*lomspb.Item) []entity.Item {
	items := make([]entity.Item, len(itemspb))

	for i := range itemspb {
		items[i] = entity.Item{
			Sku:   itemspb[i].GetSku(),
			Count: itemspb[i].GetCount(),
		}
	}

	return items
}
