package main

import (
	"context"
	"log"
	"math/rand"
	"os/signal"
	"syscall"
	"time"

	"WB_LVL0/producer/internal/config"
	"WB_LVL0/producer/internal/generator"
	"WB_LVL0/producer/internal/sender"
)

func main() {
	log.Println("Starting Order Producer Service...")

	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	orderGenerator := generator.New(random)
	orderSender := sender.New(cfg.KafkaBroker, cfg.KafkaTopic)
	defer func() {
		if err := orderSender.Close(); err != nil {
			log.Printf("close Kafka writer: %v", err)
		}
	}()

	ticker := time.NewTicker(cfg.SendInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			order, err := orderGenerator.Order()
			if err != nil {
				log.Printf("generate order: %v", err)
				continue
			}

			if err := orderSender.Send(ctx, order); err != nil {
				log.Printf("send order %s: %v", order.OrderUID, err)
				continue
			}

			log.Printf("Sent order: %s", order.OrderUID)
		case <-ctx.Done():
			log.Println("Shutting down producer...")
			return
		}
	}
}
