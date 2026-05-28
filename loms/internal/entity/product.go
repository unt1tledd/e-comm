package entity

type Item struct {
	Sku   uint32
	Count uint32
}

type ProductInfo struct {
	Name  string
	Price uint32
}

type OrderItem struct {
	Sku   uint32
	Count uint32
	Price uint32
}
