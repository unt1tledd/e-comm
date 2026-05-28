package cart

import (
	"strconv"
	"strings"

	cartpb "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/cart/api/cart/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (c *cartServer) ListCart(req *cartpb.ListCartRequest, srv cartpb.Cart_ListCartServer) error {
	if err := req.Validate(); err != nil {
		c.logger.Warn("list cart validation failed", zap.Error(err))
		return status.Errorf(codes.InvalidArgument, "%v", err)
	}

	cartInfo, err := c.cartService.ListCart(srv.Context(), req.GetUserId())

	if err != nil {
		c.logger.Error("list cart failed",
			zap.Int64("user_id", req.GetUserId()),
			zap.Error(err),
		)
		return status.Error(codes.Internal, "list cart failed")
	}

	if len(cartInfo.NotFoundItems) != 0 {
		skus := make([]string, 0, len(cartInfo.NotFoundItems))
		for _, sku := range cartInfo.NotFoundItems {
			skus = append(skus, strconv.FormatUint(uint64(sku), 10))
		}
		c.logger.Warn("Not found items with sku: " + strings.Join(skus, ", "))
	}

	if len(cartInfo.Items) != 0 {
		if err := srv.Send(entityToListCartProto(cartInfo.Items, cartInfo.TotalPrice)); err != nil {
			c.logger.Error("send list cart response failed",
				zap.Int64("user_id", req.GetUserId()),
				zap.Error(err),
			)
			return status.Error(codes.Internal, "send list cart response failed")
		}
	}

	c.logger.Info("cart listed",
		zap.Int64("user_id", req.GetUserId()),
		zap.Int("items_count", len(cartInfo.Items)),
		zap.Uint32("total_price", cartInfo.TotalPrice),
	)

	return nil
}
