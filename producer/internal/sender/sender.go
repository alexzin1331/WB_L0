package sender

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"WB_LVL0/server/models"

	"github.com/segmentio/kafka-go"
)

type Sender struct {
	writer *kafka.Writer
}

func New(broker, topic string) *Sender {
	return &Sender{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(broker),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			MaxAttempts:  3,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			Async:        true,
			Logger: kafka.LoggerFunc(func(message string, args ...interface{}) {
				log.Printf("[KAFKA] "+message, args...)
			}),
			ErrorLogger: kafka.LoggerFunc(func(message string, args ...interface{}) {
				log.Printf("[KAFKA-ERROR] "+message, args...)
			}),
			BatchSize:  100,
			BatchBytes: 1048576,
		},
	}
}

func (s *Sender) Close() error {
	return s.writer.Close()
}

func (s *Sender) Send(ctx context.Context, order models.Order) error {
	jsonData, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("marshal order: %w", err)
	}

	writeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := s.writer.WriteMessages(writeCtx, kafka.Message{
		Key:   []byte(order.OrderUID),
		Value: jsonData,
	}); err != nil {
		return fmt.Errorf("write order message: %w", err)
	}

	return nil
}
