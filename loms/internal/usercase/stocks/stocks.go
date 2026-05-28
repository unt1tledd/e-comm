package stocks

import "context"

//go:generate mockgen -source=stocks.go -destination=mocks/product_stocks.go -package=mocks

type (
	stocksRepository interface {
		GetStock(ctx context.Context, sku uint32) (uint64, error)
		SetStock(ctx context.Context, sku uint32, count uint64) error
	}

	transactor interface {
		WithTx(ctx context.Context, f func(ctx context.Context) error) error
	}
)

type stocksService struct {
	stocksRepository stocksRepository
	transactor       transactor
}

func NewStocksService(
	stocksRepository stocksRepository,
	transactor transactor,
) *stocksService {
	return &stocksService{
		stocksRepository: stocksRepository,
		transactor:       transactor,
	}
}

func (s *stocksService) GetStock(ctx context.Context, sku uint32) (uint64, error) {
	var count uint64
	err := s.transactor.WithTx(ctx, func(ctx context.Context) (err error) {
		count, err = s.stocksRepository.GetStock(ctx, sku)
		return err
	})

	if err != nil {
		return 0, err
	}
	return count, nil
}

func (s *stocksService) SetStock(ctx context.Context, sku uint32, count uint64) error {
	return s.transactor.WithTx(ctx, func(ctx context.Context) error {
		return s.stocksRepository.SetStock(ctx, sku, count)
	})
}
