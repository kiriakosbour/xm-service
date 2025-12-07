package config

import (
	"os"
)

type Config struct {
	DBUrl        string
	KafkaBrokers []string
	ServerPort   string
}

func Load() *Config {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "postgres://user:pass@localhost:5432/xm?sslmode=disable"
	}

	kafkaAddr := os.Getenv("KAFKA_BROKERS")
	if kafkaAddr == "" {
		kafkaAddr = "localhost:9092"
	}

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = ":8080"
	}

	return &Config{
		DBUrl:        dbURL,
		KafkaBrokers: []string{kafkaAddr},
		ServerPort:   port,
	}
}
