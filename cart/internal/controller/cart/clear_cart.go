package cart

import (
	"context"

	cartpb "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/cart/api/cart/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *cartServer) ClearCart(ctx context.Context, req *cartpb.ClearCartRequest) (*emptypb.Empty, error) {
	if err := req.Validate(); err != nil {
		s.logger.Warn("clear cart validation failed", zap.Error(err))
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	userID := req.GetUserId()

	err := s.cartService.ClearCart(ctx, userID)
	if err != nil {
		s.logger.Error("clear cart failed",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		return nil, status.Error(codes.Internal, "clear cart failed")
	}

	s.logger.Info("cart cleared", zap.Int64("user_id", userID))

	return &emptypb.Empty{}, nil
}
