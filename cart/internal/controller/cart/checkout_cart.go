package cart

import (
	"context"

	cartpb "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/cart/api/cart/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *cartServer) CheckoutCart(ctx context.Context, req *cartpb.CheckoutCartRequest) (*cartpb.CheckoutCartResponse, error) {
	if err := req.Validate(); err != nil {
		s.logger.Warn("checkout cart validation failed", zap.Error(err))
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	userID := req.GetUserId()

	orderID, err := s.cartService.CheckoutCart(ctx, userID)
	if err != nil {
		s.logger.Error("checkout cart failed",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		return nil, status.Error(codes.Internal, "checkout cart failed")
	}

	s.logger.Info("cart checked out",
		zap.Int64("user_id", userID),
		zap.Int64("order_id", orderID),
	)

	return &cartpb.CheckoutCartResponse{OrderId: orderID}, nil
}
