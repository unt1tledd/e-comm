package cart

import (
	"context"

	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/entity"
	cartpb "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/cart/api/cart/v1"
	"go.uber.org/zap"
)

//go:generate mockgen -source=cart.go -destination=mocks/cart_mocks.go -package=mocks

type (
	ItemService interface {
		AddItem(ctx context.Context, userID int64, item entity.OrderItem) error
		DeleteItem(ctx context.Context, userID int64, sku uint32) error
	}

	Service interface {
		ListCart(ctx context.Context, userID int64) (entity.CartInfo, error)
		ClearCart(ctx context.Context, userID int64) error
		CheckoutCart(ctx context.Context, userID int64) (orderID int64, err error)
	}
)

var _ cartpb.CartServer = (*cartServer)(nil)

type cartServer struct {
	itemService ItemService
	cartService Service
	logger      *zap.Logger
}

func NewCartServer(
	itemService ItemService,
	cartService Service,
	logger *zap.Logger,
) *cartServer {
	return &cartServer{
		itemService: itemService,
		cartService: cartService,
		logger:      logger,
	}
}

func entityToListCartProto(l []entity.Product, totalPrice uint32) *cartpb.ListCartResponse {
	resp := &cartpb.ListCartResponse{TotalPrice: totalPrice}
	resp.Items = make([]*cartpb.Item, len(l))
	for i, item := range l {
		resp.Items[i] = &cartpb.Item{
			Sku:   item.Sku,
			Count: item.Count,
			Name:  item.Name,
			Price: item.Price,
		}
	}

	return resp
}
