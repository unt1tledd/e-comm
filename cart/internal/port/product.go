package port

import "errors"

var (
	ErrProductNotFound = errors.New("product not found")
)

type ProductInfo struct {
	Name  string
	Price uint32
}
