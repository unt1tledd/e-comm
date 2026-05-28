package cart

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	cartpb "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/cart/api/cart/v1"
)

func (c *cartServer) DeleteItem(ctx context.Context, req *cartpb.DeleteItemRequest) (*emptypb.Empty, error) {
	if err := req.Validate(); err != nil {
		c.logger.Warn("delete item validation failed", zap.Error(err))
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	userID := req.GetUserId()
	sku := req.GetSku()

	err := c.itemService.DeleteItem(ctx, userID, sku)
	if err != nil {
		c.logger.Error("delete item failed",
			zap.Int64("user_id", userID),
			zap.Uint32("sku", sku),
			zap.Error(err),
		)
		return nil, status.Error(codes.Internal, "delete item failed")
	}

	c.logger.Info("item deleted from cart",
		zap.Int64("user_id", userID),
		zap.Uint32("sku", sku),
	)

	return &emptypb.Empty{}, nil
}
