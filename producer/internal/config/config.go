package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

const (
	defaultKafkaBroker  = "kafka:9092"
	defaultKafkaTopic   = "orders"
	defaultSendInterval = 5 * time.Second
)

type Config struct {
	KafkaBroker  string
	KafkaTopic   string
	SendInterval time.Duration
}

func Load() Config {
	return Config{
		KafkaBroker:  envString("KAFKA_BROKER", defaultKafkaBroker),
		KafkaTopic:   envString("KAFKA_TOPIC", defaultKafkaTopic),
		SendInterval: envDurationSeconds("PRODUCER_SEND_INTERVAL_SECONDS", defaultSendInterval),
	}
}

func envString(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envDurationSeconds(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		log.Printf("invalid %s=%q, using %s", key, value, fallback)
		return fallback
	}

	return time.Duration(seconds) * time.Second
}
