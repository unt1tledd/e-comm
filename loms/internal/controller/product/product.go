package product

import (
	"context"
	"errors"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	xerror "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	productpb "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/product/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//go:generate mockgen -source=product.go -destination=mocks/product_mocks.go -package=mocks

type (
	productService interface {
		CreateProduct(ctx context.Context, name string, price uint32) (uint32, error)
		GetProduct(ctx context.Context, sku uint32) (*entity.ProductInfo, error)
	}
)

type productServer struct {
	productService productService
}

func NewProductServer(productService productService) *productServer {
	return &productServer{
		productService: productService,
	}
}

func (s *productServer) GetProduct(ctx context.Context, req *productpb.GetProductRequest) (*productpb.GetProductResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	item, err := s.productService.GetProduct(ctx, req.GetSku())
	if err != nil {
		switch {
		case errors.Is(err, xerror.ErrNotFoundProduct):
			return nil, status.Errorf(codes.NotFound, "%v", err)
		default:
			return nil, status.Error(codes.Internal, "get product failed")
		}
	}

	return &productpb.GetProductResponse{
		Name:  item.Name,
		Price: item.Price,
	}, nil
}

func (s *productServer) CreateProduct(ctx context.Context, req *productpb.CreateProductRequest) (*productpb.CreateProductResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	sku, err := s.productService.CreateProduct(ctx, req.GetName(), req.GetPrice())
	if err != nil {
		return nil, status.Error(codes.Internal, "create product failed")
	}

	return &productpb.CreateProductResponse{Sku: sku}, nil
}
