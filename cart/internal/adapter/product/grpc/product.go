package grpc

import (
	"context"

	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/port"
	pb "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/product/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type productClient struct {
	client pb.ProductServiceClient
}

func NewProductClient(
	product pb.ProductServiceClient,
) *productClient {
	return &productClient{
		client: product,
	}
}

func (c *productClient) GetProductInfo(ctx context.Context, sku uint32) (*port.ProductInfo, error) {
	resp, err := c.client.GetProduct(ctx, &pb.GetProductRequest{Sku: sku})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				return nil, port.ErrProductNotFound
			default:
				return nil, err
			}
		}
		return nil, err
	}

	return &port.ProductInfo{
		Name:  resp.GetName(),
		Price: resp.GetPrice(),
	}, nil
}
