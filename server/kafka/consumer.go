package kafka

import (
	"WB_LVL0/server/internal/storage"
	"WB_LVL0/server/models"
	"context"
	"encoding/json"
	"fmt"
	"github.com/segmentio/kafka-go"
	"log"
	"math"
	"math/rand"
	"time"
)

const (
	//kafkaBroker  = "localhost:9092" -- local

	kafkaBroker     = "kafka:9092"
	kafkaTopic      = "orders"
	kafkaDlqTopic   = "orders_dlq"
	kafkaGroupID    = "order-consumers"
	maxRetryAttempt = 5
	initialBackoff  = 100 * time.Millisecond
	maxBackoff      = 5 * time.Second
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
	dlqWriter := NewDLQWriter()
	defer dlqWriter.Close()

	for {
		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Failed to read message: %v", err)
			continue
		}

		if err := processWithRetry(db, dlqWriter, msg); err != nil {
			log.Printf("Failed to process message after retries, moved to DLQ: %v", err)
		}
	}
}

func processWithRetry(db *storage.Storage, dlqWriter *kafka.Writer, msg kafka.Message) error {
	var lastErr error

	for attempt := 0; attempt < maxRetryAttempt; attempt++ {
		if attempt > 0 {
			backoff := calculateBackoff(attempt)
			log.Printf("Retry attempt %d/%d after %v for message offset=%d",
				attempt, maxRetryAttempt, backoff, msg.Offset)
			time.Sleep(backoff)
		}

		err := processMessage(db, msg)
		if err == nil {
			return nil // Success
		}

		lastErr = err
		log.Printf("Attempt %d/%d failed: %v", attempt+1, maxRetryAttempt, err)

		// Don't retry for validation errors
		if _, ok := err.(*models.ValidationError); ok {
			break
		}
	}

	// All retries failed, send to DLQ
	if err := sendToDLQ(dlqWriter, msg, lastErr); err != nil {
		return fmt.Errorf("failed to send to DLQ: %w (original error: %v)", err, lastErr)
	}

	return lastErr
}

func calculateBackoff(attempt int) time.Duration {
	// Exponential backoff with jitter
	backoff := float64(initialBackoff) * math.Pow(2, float64(attempt))
	if backoff > float64(maxBackoff) {
		backoff = float64(maxBackoff)
	}

	// Add jitter to avoid thundering herd
	jitter := rand.Float64() * (backoff * 0.2) // ±20% jitter
	backoff = backoff - (backoff * 0.1) + jitter

	return time.Duration(backoff)
}

func sendToDLQ(writer *kafka.Writer, msg kafka.Message, processingErr error) error {
	dlqMessage := struct {
		OriginalMessage kafka.Message
		Error           string
		Timestamp       time.Time
	}{
		OriginalMessage: msg,
		Error:           processingErr.Error(),
		Timestamp:       time.Now(),
	}

	dlqData, err := json.Marshal(dlqMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal DLQ message: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return writer.WriteMessages(ctx, kafka.Message{
		Key:   msg.Key,
		Value: dlqData,
	})
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
