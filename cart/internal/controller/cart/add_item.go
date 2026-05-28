package cart

import (
	"context"
	"errors"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/entity"
	xerror "github.com/igoroutine-courses/microservices.ecommerce.cart/internal/errors"
	cartpb "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/cart/api/cart/v1"
)

func (s *cartServer) AddItem(ctx context.Context, req *cartpb.AddItemRequest) (*emptypb.Empty, error) {
	if err := req.Validate(); err != nil {
		s.logger.Warn("add item validation failed", zap.Error(err))
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	userID := req.GetUserId()
	item := entity.OrderItem{
		Sku:   req.GetSku(),
		Count: req.GetCount(),
	}

	if err := s.itemService.AddItem(ctx, userID, item); err != nil {
		switch {
		case errors.Is(err, xerror.ErrProductNotFound):
			s.logger.Warn("add item product not found",
				zap.Int64("user_id", userID),
				zap.Uint32("sku", item.Sku),
				zap.Error(err),
			)
			return nil, status.Errorf(codes.NotFound, "%v", err)
		case errors.Is(err, xerror.ErrInsufficientStock):
			s.logger.Warn("add item insufficient stock",
				zap.Int64("user_id", userID),
				zap.Uint32("sku", item.Sku),
				zap.Uint32("count", item.Count),
				zap.Error(err),
			)
			return nil, status.Errorf(codes.FailedPrecondition, "%v", err)
		default:
			s.logger.Error("add item failed",
				zap.Int64("user_id", userID),
				zap.Uint32("sku", item.Sku),
				zap.Uint32("count", item.Count),
				zap.Error(err),
			)
			return nil, status.Error(codes.Internal, "add item failed")
		}
	}

	s.logger.Info("item added to cart",
		zap.Int64("user_id", userID),
		zap.Uint32("sku", item.Sku),
		zap.Uint32("count", item.Count),
	)

	return &emptypb.Empty{}, nil
}
