package stocks

import (
	"context"
	"errors"

	xerror "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	stockpb "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/stocks/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

//go:generate mockgen -source=stocks.go -destination=mocks/stocks_mocks.go -package=mocks

type (
	stocksService interface {
		GetStock(ctx context.Context, sku uint32) (uint64, error)
		SetStock(ctx context.Context, sku uint32, count uint64) error
	}
)

type stocksServer struct {
	stocksService stocksService
}

func NewStocksServer(stocksService stocksService) *stocksServer {
	return &stocksServer{
		stocksService: stocksService,
	}
}

func (s *stocksServer) GetStock(ctx context.Context, req *stockpb.GetStockRequest) (*stockpb.GetStockResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	count, err := s.stocksService.GetStock(ctx, req.GetSku())
	if err != nil {
		switch {
		case errors.Is(err, xerror.ErrNotFoundProduct):
			return nil, status.Errorf(codes.NotFound, "%v", err)
		default:
			return nil, status.Error(codes.Internal, "get stock failed")
		}
	}

	return &stockpb.GetStockResponse{Count: count}, nil
}

func (s *stocksServer) SetStock(ctx context.Context, req *stockpb.SetStockRequest) (*emptypb.Empty, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	if err := s.stocksService.SetStock(ctx, req.GetSku(), req.GetCount()); err != nil {
		switch {
		case errors.Is(err, xerror.ErrNotFoundProduct):
			return nil, status.Errorf(codes.NotFound, "%v", err)
		default:
			return nil, status.Error(codes.Internal, "set stock failed")
		}
	}

	return &emptypb.Empty{}, nil
}
