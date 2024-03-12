package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	_ "github.com/lib/pq"
	stan "github.com/nats-io/stan.go"
)

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

// Подключение к кластеру NATS Streaming
func NatsConnection(clientID string) stan.Conn {
	sc, err := stan.Connect("test-cluster", clientID, stan.NatsURL("nats://localhost:4222"))
	if err != nil {
		log.Fatalf("Ошибка подключения к NATS Streaming: %v", err)
	}
	return sc
}

// Открытие соединения с базой данных
func DatabaseConnection() *sql.DB {
	db, err := sql.Open("postgres", "host=localhost port=5438 user=segadron password=12345 dbname=Wildberries sslmode=disable")
	if err != nil {
		log.Fatalf("Ошибка подключения к PostgreSQL: %v", err)
	}
	return db
}

func main() {
	// Подключаемся к кластеру NATS Streaming
	sc := NatsConnection("test-publisher-id")

	// Открываем соединение с базой данных
	DatabaseConnection()

	// Генерируем и отправляем случайные заказы на сервер
	ordersSending(sc)
}

// Генерация и отправка случайных заказов на сервер
func ordersSending(sc stan.Conn) {
	for i := 0; i < 10; i++ {
		order := generateRandomOrder()
		jsonData, err := json.Marshal(order)
		if err != nil {
			fmt.Println("Error while marshalling JSON...")
		}

		pubErr := sc.Publish("test-channel", jsonData)
		if pubErr != nil {
			fmt.Println("Проблема с отправкой данных...")
		} else {
			fmt.Printf("Данные отправлены...")
		}

		// Задержка между отправкой заказов
		time.Sleep(time.Second)
	}
}

// Генерация случайных данных для заказа
func generateRandomOrder() Order {
	orderUID := randomGenerator(19)
	trackNumber := randomGenerator(14)
	entry := randomGenerator(4)
	locale := "en"
	internalSignature := ""
	customerID := "test"
	deliveryService := "meest"
	shardKey := "9"
	smID := 99
	dateCreated := time.Now().UTC()
	oofShard := "1"

	delivery := Delivery{
		Name:    randomGenerator(11),
		Phone:   randomGenerator(11),
		Zip:     "2639809",
		City:    "Kiryat Mozkin",
		Address: "Ploshad Mira 15",
		Region:  "Kraiot",
		Email:   "test@gmail.com",
	}

	payment := Payment{
		Transaction:  orderUID,
		RequestID:    "",
		Currency:     "USD",
		Provider:     "wbpay",
		Amount:       rand.Intn(2000) + 1,
		PaymentDt:    time.Now().Unix(),
		Bank:         "alpha",
		DeliveryCost: 1500,
		GoodsTotal:   317,
		CustomFee:    0,
	}

	items := []Item{
		{
			ChrtID:      rand.Intn(9999999) + 1,
			TrackNumber: trackNumber,
			Price:       rand.Intn(2000) + 100,
			Rid:         randomGenerator(21),
			Name:        randomGenerator(8),
			Sale:        30,
			Size:        "0",
			TotalPrice:  317,
			NmID:        rand.Intn(9999999) + 1,
			Brand:       randomGenerator(13),
			Status:      202,
		},
	}

	return Order{
		OrderUID:          orderUID,
		TrackNumber:       trackNumber,
		Entry:             entry,
		Delivery:          delivery,
		Payment:           payment,
		Items:             items,
		Locale:            locale,
		InternalSignature: internalSignature,
		CustomerID:        customerID,
		DeliveryService:   deliveryService,
		ShardKey:          shardKey,
		SmID:              smID,
		DateCreated:       dateCreated,
		OofShard:          oofShard,
	}
}

func randomGenerator(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}
