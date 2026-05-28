package controller

import (
	lomspb "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/loms/v1"
	productpb "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/product/v1"
	stockspb "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/stocks/v1"
	"google.golang.org/grpc"
)

type API struct {
	lomsServer    lomspb.LomsServer
	productServer productpb.ProductServiceServer
	stocksServer  stockspb.StocksServer
}

func New(
	lomsServer lomspb.LomsServer,
	productServer productpb.ProductServiceServer,
	stocksServer stockspb.StocksServer,
) *API {
	return &API{
		lomsServer:    lomsServer,
		productServer: productServer,
		stocksServer:  stocksServer,
	}
}

func (a *API) Register(s *grpc.Server) {
	lomspb.RegisterLomsServer(s, a.lomsServer)
	productpb.RegisterProductServiceServer(s, a.productServer)
	stockspb.RegisterStocksServer(s, a.stocksServer)
}
