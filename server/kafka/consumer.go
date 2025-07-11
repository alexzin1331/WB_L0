package kafka

import (
	"WB_LVL0/server/internal/storage"
	"WB_LVL0/server/models"
	"context"
	"encoding/json"
	"fmt"
	"github.com/segmentio/kafka-go"
	"log"
	"time"
)

const (
	//kafkaBroker     = "localhost:9092" -- local
	kafkaBroker  = "kafka:9092"
	kafkaTopic   = "orders"
	kafkaGroupID = "order-consumers"
)

func NewReader() *kafka.Reader {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{kafkaBroker},
		Topic:       kafkaTopic,
		GroupID:     kafkaGroupID,
		MinBytes:    10e3, // 10KB
		MaxBytes:    10e6, // 10MB
		StartOffset: kafka.FirstOffset,
		Logger: kafka.LoggerFunc(func(s string, args ...interface{}) {
			log.Printf("[KAFKA-CONSUMER] "+s, args...)
		}),
		ErrorLogger: kafka.LoggerFunc(func(s string, args ...interface{}) {
			log.Printf("[KAFKA-CONSUMER-ERROR] "+s, args...)
		}),
	})
	return reader
}

func NewDLQWriter() *kafka.Writer {
	return &kafka.Writer{
		Addr:         kafka.TCP(kafkaBroker),
		Topic:        kafkaDlqTopic,
		Balancer:     &kafka.Hash{},
		MaxAttempts:  3,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		Logger: kafka.LoggerFunc(func(s string, args ...interface{}) {
			log.Printf("[KAFKA-DLQ] "+s, args...)
		}),
		ErrorLogger: kafka.LoggerFunc(func(s string, args ...interface{}) {
			log.Printf("[KAFKA-DLQ-ERROR] "+s, args...)
		}),
	}
}

// ReadMSG listens for Kafka messages and processes them with retry and DLQ
func ReadMSG(db *storage.Storage, reader *kafka.Reader) {
	for {
		//get messages
		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Failed to read message: %v", err)
			continue
		}
		//pass messages for processing
		if err := processMessage(db, msg); err != nil {
			log.Printf("Failed to process message: %v", err)
		}
	}
}

func processMessage(db *storage.Storage, msg kafka.Message) error {
	startTime := time.Now()
	log.Printf("Processing message: offset=%d partition=%d", msg.Offset, msg.Partition)

	var order models.Order
	if err := json.Unmarshal(msg.Value, &order); err != nil {
		return fmt.Errorf("failed to unmarshal order: %w", err)
	}

	// validate data
	if err := order.Validate(); err != nil {
		return fmt.Errorf("invalid order data: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// save to PostgreSQL and redis
	if err := db.SaveOrder(ctx, order); err != nil {
		return fmt.Errorf("failed to save order: %w", err)
	}

	log.Printf(
		"Order processed successfully: order_uid=%s items=%d time=%v",
		order.OrderUID,
		len(order.Items),
		time.Since(startTime),
	)

	return nil
}
