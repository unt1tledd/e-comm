package entity

type Product struct {
	Sku   uint32 `json:"sku"`
	Count uint32 `json:"count"`
	Name  string `json:"name"`
	Price uint32 `json:"price"`
}
