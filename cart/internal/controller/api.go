package controller

import cartpb "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/cart/api/cart/v1"

type API struct {
	cartServer cartpb.CartServer
}

func New(cartServer cartpb.CartServer) *API {
	return &API{
		cartServer: cartServer,
	}
}

func (a *API) GetSrv() cartpb.CartServer {
	return a.cartServer
}
