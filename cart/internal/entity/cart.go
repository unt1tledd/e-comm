package entity

type Cart struct {
	Items []OrderItem `json:"items"`
}

type OrderItem struct {
	Sku   uint32 `json:"sku"`
	Count uint32 `json:"count"`
}

type CartInfo struct {
	Items         []Product `json:"items"`
	TotalPrice    uint32    `json:"total_price"`
	NotFoundItems []uint32  `json:"not_found_items"`
}
