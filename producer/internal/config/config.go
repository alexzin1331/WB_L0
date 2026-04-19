package config

import (
	"os"
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
		SendInterval: defaultSendInterval,
	}
}

func envString(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
