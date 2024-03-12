package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	_ "github.com/lib/pq"
	stan "github.com/nats-io/stan.go"
)

// Структура для работы с базой данных
type Order struct {
	OrderUID  string
	OrderJSON json.RawMessage
}

// Отображение для кэширования данных
var cache map[string]*Order

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

// Восстановление кэша из базы
func cacheRestore(db *sql.DB, cache map[string]*Order) {
	// Получение всех orderUID из базы данных
	rows, err := db.Query("SELECT order_uid FROM \"WB\".orders")
	if err != nil {
		fmt.Printf("ошибка при выполнении запроса: %v\n", err)
		return
	}
	defer rows.Close()

	var orderUIDs []string

	for rows.Next() {
		var orderUID string
		if err := rows.Scan(&orderUID); err != nil {
			fmt.Printf("ошибка при сканировании данных: %v\n", err)
			return
		}
		orderUIDs = append(orderUIDs, orderUID)
	}

	if err := rows.Err(); err != nil {
		fmt.Printf("ошибка при получении данных из базы данных: %v\n", err)
		return
	}

	// Итерирование по массиву orderUIDs и кэширование каждого значения
	for _, orderUID := range orderUIDs {
		getOrderData(cache, orderUID, db)
	}
}

// Кэширование данных заказа
func getOrderData(cache map[string]*Order, orderUID string, db *sql.DB) *Order {
	order, err := getFromCache(cache, orderUID)
	if err != nil {
		// Если значение отсутствует в кэше, получаем его из базы данных
		order, err = getFromDB(db, orderUID)
		if err != nil {
			log.Printf("ошибка при получении данных из базы данных: %v\n", err)
			return nil
		}
		// Кэшируем значение
		err = setToCache(cache, order)
		if err != nil {
			log.Printf("ошибка при кэшировании данных: %v\n", err)
			return nil
		}
	}
	return order
}

// Получение данных заказа из кэша
func getFromCache(cache map[string]*Order, orderUID string) (*Order, error) {
	order, ok := cache[orderUID]
	if !ok {
		return nil, fmt.Errorf("данные с ключом %s не найдены в кэше", orderUID)
	}
	return order, nil
}

// Получение данных заказа из базы
func getFromDB(db *sql.DB, orderUID string) (*Order, error) {
	row := db.QueryRow("SELECT data FROM \"WB\".orders WHERE order_uid = $1", orderUID)

	var orderJSON json.RawMessage
	err := row.Scan(&orderJSON)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении данных из базы данных: %v", err)
	}

	order := &Order{
		OrderUID:  orderUID,
		OrderJSON: orderJSON,
	}

	return order, nil
}

// Запись данных заказа в кэш
func setToCache(cache map[string]*Order, order *Order) error {
	cache[order.OrderUID] = order
	return nil
}

// Подписка на канал NATS Streaming
func channelSubscribtion(sc stan.Conn, db *sql.DB) {
	handler := func(msg *stan.Msg) {
		insertingOrders(db, msg)
	}

	_, err := sc.Subscribe("test-channel", handler, stan.DurableName("my-durable"))
	if err != nil {
		log.Fatalf("Ошибка подписки на канал: %v", err)
	}
}

// Запись полученных данных в базу
func insertingOrders(db *sql.DB, msg *stan.Msg) {
	var order Order
	err := json.Unmarshal(msg.Data, &order)
	if err != nil {
		fmt.Printf("ошибка при распаковке сообщения: %v", err)
	}

	// Выполняем SQL-запрос для вставки данных в базу данных
	_, err = db.Exec("INSERT INTO \"WB\".orders (order_uid, data) VALUES ($1, $2)", order.OrderUID, msg.Data)
	if err != nil {
		fmt.Printf("ошибка при выполнении SQL-запроса: %v", err)
	} else {
		fmt.Println("Данные занесены в базу")
	}
}

// Запуск HTTP-сервера
func serverStart(cache map[string]*Order, db *sql.DB) {
	//Задать стартовую страницу http://localhost:8080
	http.Handle("/", http.FileServer(http.Dir("frontend")))
	// Обработчик для запроса данных заказа по его уникальному идентификатору
	http.HandleFunc("/order/", func(w http.ResponseWriter, r *http.Request) {
		// Получаем orderUID из URL запроса
		orderUID := r.URL.Path[len("/order/"):]

		order := getOrderData(cache, orderUID, db)

		// Преобразуем данные заказа в JSON
		orderJSON, err := json.Marshal(order)
		if err != nil {
			log.Printf("ошибка при преобразовании данных заказа в JSON: %v\n", err)
			http.Error(w, "Ошибка при преобразовании данных в JSON", http.StatusInternalServerError)
			return
		}

		// Отправляем данные заказа в качестве ответа
		w.Header().Set("Content-Type", "application/json")
		w.Write(orderJSON)
	})

	log.Println("Сервер запущен : http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("Ошибка при запуске сервера:", err)
	}
}

func main() {
	// Подключаемся к кластеру NATS Streaming
	sc := NatsConnection("test-client-id")

	// Открываем соединение с базой данных
	db := DatabaseConnection()

	// Создаем экземпляр кэша
	cache = make(map[string]*Order)

	// Восстанавливаем кэш из базы
	cacheRestore(db, cache)

	// Делаем подписку на NATS-канал
	channelSubscribtion(sc, db)

	// Запускаем сервер
	serverStart(cache, db)
}
