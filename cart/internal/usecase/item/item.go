package item

import (
	"context"
	"errors"

	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/entity"
	xerror "github.com/igoroutine-courses/microservices.ecommerce.cart/internal/errors"
	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/port"
)

//go:generate mockgen -source=item.go -destination=mocks/item_mocks.go -package=mocks

type (
	cartRepository interface {
		AddItem(ctx context.Context, userID int64, item entity.OrderItem) error
		DeleteItem(ctx context.Context, userID int64, sku uint32) error
		GetItemsByUserID(ctx context.Context, userID int64) ([]entity.OrderItem, error)
	}

	productClient interface {
		GetProductInfo(ctx context.Context, sku uint32) (*port.ProductInfo, error)
	}

	lomsClient interface {
		GetStock(ctx context.Context, sku uint32) (uint64, error)
	}
)

type itemService struct {
	cartRepository cartRepository
	productClient  productClient
	lomsClient     lomsClient
}

func NewItemService(
	cartRepository cartRepository,
	productClient productClient,
	lomsCLient lomsClient,
) *itemService {
	return &itemService{
		cartRepository: cartRepository,
		productClient:  productClient,
		lomsClient:     lomsCLient,
	}
}

func (s *itemService) AddItem(ctx context.Context, userID int64, item entity.OrderItem) error {
	if _, err := s.productClient.GetProductInfo(ctx, item.Sku); err != nil {
		switch {
		case errors.Is(err, port.ErrProductNotFound):
			return xerror.NewProductNotFoundError(item.Sku)
		default:
			return err
		}
	}

	stockCnt, err := s.lomsClient.GetStock(ctx, item.Sku)
	if err != nil {
		return err
	}

	if stockCnt < uint64(item.Count) {
		return xerror.ErrInsufficientStock
	}

	if err := s.cartRepository.AddItem(ctx, userID, item); err != nil {
		return err
	}

	return nil
}

func (s *itemService) DeleteItem(ctx context.Context, userID int64, sku uint32) error {
	return s.cartRepository.DeleteItem(ctx, userID, sku)
}
