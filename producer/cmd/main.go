package main

import (
	"WB_LVL0/server/models"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-faker/faker/v4"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	//kafkaBroker  = "localhost:9092" -- local
	kafkaBroker  = "kafka:9092"
	kafkaTopic   = "orders"
	sendInterval = 5 * time.Second
)

type Address struct {
	City    string
	Address string
	Region  string
}

func main() {
	fmt.Println("Starting Order Producer Service...")
	r := rand.New(rand.NewSource(time.Now().Unix()))
	writer := &kafka.Writer{
		Addr:         kafka.TCP(kafkaBroker),
		Topic:        kafkaTopic,
		Balancer:     &kafka.LeastBytes{},
		MaxAttempts:  3,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		Async:        true,
		Logger: kafka.LoggerFunc(func(s string, args ...interface{}) {
			log.Printf("[KAFKA] "+s, args...)
		}),
		ErrorLogger: kafka.LoggerFunc(func(s string, args ...interface{}) {
			log.Printf("[KAFKA-ERROR] "+s, args...)
		}),
		BatchSize:  100,
		BatchBytes: 1048576, //1MB
	}
	defer writer.Close()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Message generation loop
	ticker := time.NewTicker(sendInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			order := generateRandomOrder(r)
			if err := sendOrder(writer, order); err != nil {
				fmt.Printf("Error sending order: %v\n", err)
			} else {
				fmt.Printf("Sent order: %s\n", order.OrderUID)
			}

		case <-quit:
			fmt.Println("Shutting down producer...")
			return
		}
	}
}

// send data to consumer
func sendOrder(writer *kafka.Writer, order models.Order) error {
	jsonData, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshal order: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(order.OrderUID),
		Value: jsonData,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return writer.WriteMessages(ctx, msg)
}

// generate random data for testing
func generateRandomOrder(r *rand.Rand) models.Order {
	// Generate unique order ID
	orderUID := uuid.New().String()

	// Generate random items
	itemCount := rand.Intn(3) + 1 // 1-3 items
	items := make([]models.Item, itemCount)
	for i := 0; i < itemCount; i++ {
		items[i] = models.Item{
			ChrtID:      r.Intn(10000000),
			TrackNumber: fmt.Sprintf("TRK%06d", r.Intn(1000000)),
			Price:       r.Intn(1000) + 100,
			Rid:         uuid.New().String(),
			Name:        faker.Word(),
			Sale:        r.Intn(50),
			Size:        fmt.Sprintf("%d", r.Intn(10)),
			TotalPrice:  r.Intn(500) + 50,
			NmID:        r.Intn(10000000),
			Brand:       faker.FirstName() + " " + faker.LastName(),
			Status:      200 + r.Intn(3),
		}
	}

	// Generate random names and addresses
	fullName := faker.FirstName() + " " + faker.LastName()
	email := faker.Email()

	address := Address{}
	if err := faker.FakeData(&address); err != nil {
		log.Fatal("can't create fake address")
	}
	return models.Order{
		OrderUID:    orderUID,
		TrackNumber: fmt.Sprintf("WBIL%08d", r.Intn(100000000)),
		Entry:       "WBIL",
		Delivery: models.Delivery{
			Name:    fullName,
			Phone:   "+" + fmt.Sprintf("%d", r.Intn(9999999999)),
			Zip:     fmt.Sprintf("%d", r.Intn(99999)),
			City:    address.City,
			Address: address.Address,
			Region:  address.Region,
			Email:   email,
		},
		Payment: models.Payment{
			Transaction:  orderUID,
			RequestID:    "",
			Currency:     "USD",
			Provider:     "wbpay",
			Amount:       r.Intn(10000) + 1000,
			PaymentDt:    time.Now().Unix(),
			Bank:         []string{"alpha", "sber", "tinkoff"}[r.Intn(3)],
			DeliveryCost: r.Intn(2000) + 500,
			GoodsTotal:   r.Intn(500) + 100,
			CustomFee:    0,
		},
		Items:             items,
		Locale:            []string{"en", "ru"}[r.Intn(2)],
		InternalSignature: "",
		CustomerID:        fmt.Sprintf("user%d", r.Intn(1000)),
		DeliveryService:   []string{"meest", "russianpost", "dhl"}[r.Intn(3)],
		Shardkey:          fmt.Sprintf("%d", r.Intn(10)),
		SmID:              r.Intn(100),
		DateCreated:       time.Now(),
		OofShard:          fmt.Sprintf("%d", rand.Intn(5)+1),
	}
}
