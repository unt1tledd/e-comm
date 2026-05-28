package product

import (
	"context"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	xerror "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
)

//go:generate mockgen -source=product.go -destination=mocks/product_mocks.go -package=mocks

type (
	productRepository interface {
		CreateProduct(ctx context.Context, product *entity.ProductInfo) (uint32, error)
		GetProduct(ctx context.Context, sku uint32) (*entity.ProductInfo, error)
	}

	stocksRepository interface {
		CreateStock(ctx context.Context, sku uint32) error
	}

	transactor interface {
		WithTx(ctx context.Context, f func(ctx context.Context) error) error
	}
)

type productService struct {
	productRepository productRepository
	stocksRepository  stocksRepository
	transactor        transactor
}

func NewProductService(
	productRepository productRepository,
	stocksRepository stocksRepository,
	transactor transactor,
) *productService {
	return &productService{
		productRepository: productRepository,
		stocksRepository:  stocksRepository,
		transactor:        transactor,
	}
}

func (s *productService) CreateProduct(ctx context.Context, name string, price uint32) (uint32, error) {
	var sku uint32

	err := s.transactor.WithTx(ctx, func(ctx context.Context) (err error) {
		product := entity.ProductInfo{Name: name, Price: price}
		sku, err = s.productRepository.CreateProduct(ctx, &product)
		if err != nil {
			return err
		}

		if err := s.stocksRepository.CreateStock(ctx, sku); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	return sku, nil
}

func (s *productService) GetProduct(ctx context.Context, sku uint32) (*entity.ProductInfo, error) {
	product, err := s.productRepository.GetProduct(ctx, sku)
	if err != nil {
		return nil, xerror.ErrNotFoundProduct
	}
	return product, nil
}
