package structures

import "time"

// Структуры данных для генерации данных заказов
type Order struct {
	OrderUID          string
	TrackNumber       string
	Entry             string
	Delivery          Delivery
	Payment           Payment
	Items             []Item
	Locale            string
	InternalSignature string
	CustomerID        string
	DeliveryService   string
	ShardKey          string
	SmID              int
	DateCreated       time.Time
	OofShard          string
}

type Delivery struct {
	DeliveryID int
	Name       string
	Phone      string
	Zip        string
	City       string
	Address    string
	Region     string
	Email      string
}

type Item struct {
	ChrtID      int
	TrackNumber string
	Price       int
	Rid         string
	Name        string
	Sale        int
	Size        string
	TotalPrice  int
	NmID        int
	Brand       string
	Status      int
}

type Payment struct {
	Transaction  string
	RequestID    string
	Currency     string
	Provider     string
	Amount       int
	PaymentDt    int64
	Bank         string
	DeliveryCost int
	GoodsTotal   int
	CustomFee    int
}
