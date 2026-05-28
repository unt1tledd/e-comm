package grpc

import (
	"context"

	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/port"
	lomspb "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/loms/v1"
	stockspb "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/stocks/v1"
)

type lomsClient struct {
	client      lomspb.LomsClient
	stockClient stockspb.StocksClient
}

func NewLOMSClient(
	loms lomspb.LomsClient,
	stockClient stockspb.StocksClient,
) *lomsClient {
	return &lomsClient{
		client:      loms,
		stockClient: stockClient,
	}
}

func (c *lomsClient) GetStock(ctx context.Context, sku uint32) (uint64, error) {
	resp, err := c.stockClient.GetStock(ctx, &stockspb.GetStockRequest{Sku: sku})
	if err != nil {
		return 0, err
	}

	return resp.GetCount(), nil
}

func (c *lomsClient) CreateOrder(ctx context.Context, userID int64, items []port.Item) (int64, error) {
	lomsItems := make([]*lomspb.Item, len(items))

	for i, item := range items {
		lomsItems[i] = &lomspb.Item{
			Sku:   item.SKU,
			Count: item.Count,
		}
	}

	resp, err := c.client.CreateOrder(ctx, &lomspb.CreateOrderRequest{
		UserId: userID,
		Items:  lomsItems,
	})

	if err != nil {
		return 0, err
	}

	return resp.GetOrderId(), nil
}
