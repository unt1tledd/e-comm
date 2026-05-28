package cart

import (
	"context"

	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/entity"
	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/port"
)

//go:generate mockgen -source=cart.go -destination=mocks/cart_mocks.go -package=mocks

type (
	cartRepository interface {
		GetItemsByUserID(ctx context.Context, userID int64) ([]entity.OrderItem, error)
		DeleteItem(ctx context.Context, userID int64, sku uint32) error
		ClearCart(ctx context.Context, userID int64) error
	}

	productClient interface {
		GetProductInfo(ctx context.Context, sku uint32) (*port.ProductInfo, error)
	}

	lomsClient interface {
		CreateOrder(ctx context.Context, userID int64, items []port.Item) (int64, error)
	}
)

type cartService struct {
	cartRepository cartRepository
	productClient  productClient
	lomsClient     lomsClient
}

func NewCartService(
	cartRepository cartRepository,
	productClient productClient,
	lomsClient lomsClient,
) *cartService {
	return &cartService{
		cartRepository: cartRepository,
		productClient:  productClient,
		lomsClient:     lomsClient,
	}
}

func (s *cartService) CheckoutCart(ctx context.Context, userID int64) (int64, error) {
	items, err := s.cartRepository.GetItemsByUserID(ctx, userID)
	if err != nil {
		return 0, err
	}
	portItems := make([]port.Item, len(items))
	for i := range items {
		portItems[i] = port.Item{
			SKU:   items[i].Sku,
			Count: items[i].Count,
		}
	}

	orderID, err := s.lomsClient.CreateOrder(ctx, userID, portItems)
	if err != nil {
		return 0, err
	}

	if err := s.cartRepository.ClearCart(ctx, userID); err != nil {
		return 0, err
	}

	return orderID, nil
}

func (s *cartService) ListCart(ctx context.Context, userID int64) (entity.CartInfo, error) {
	items, err := s.cartRepository.GetItemsByUserID(ctx, userID)
	if err != nil {
		return entity.CartInfo{}, err
	}

	productItems := make([]entity.Product, 0, len(items))
	totalPrice := uint32(0)
	var notFoundItems []uint32

	for _, item := range items {
		productInfo, err := s.productClient.GetProductInfo(ctx, item.Sku)
		if err != nil {
			notFoundItems = append(notFoundItems, item.Sku)
			continue
		}

		totalPrice += productInfo.Price * item.Count

		productItems = append(productItems, entity.Product{
			Sku:   item.Sku,
			Count: item.Count,
			Name:  productInfo.Name,
			Price: productInfo.Price,
		})
	}

	return entity.CartInfo{
		Items:         productItems,
		TotalPrice:    totalPrice,
		NotFoundItems: notFoundItems,
	}, nil
}

func (s *cartService) ClearCart(ctx context.Context, userID int64) error {
	return s.cartRepository.ClearCart(ctx, userID)
}
