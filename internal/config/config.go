package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Kafka    KafkaConfig
	JWT      JWTConfig
}

// ServerConfig holds HTTP server settings
type ServerConfig struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// KafkaConfig holds Kafka settings
type KafkaConfig struct {
	Brokers []string
	Topic   string
	Enabled bool
}

// JWTConfig holds JWT settings
type JWTConfig struct {
	Secret string
}

// Load reads configuration from environment variables with sensible defaults
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:            getEnv("SERVER_PORT", ":8080"),
			ReadTimeout:     getDurationEnv("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    getDurationEnv("SERVER_WRITE_TIMEOUT", 15*time.Second),
			ShutdownTimeout: getDurationEnv("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		Database: DatabaseConfig{
			URL:             getEnv("DB_URL", "postgres://user:pass@localhost:5432/xm?sslmode=disable"),
			MaxOpenConns:    getIntEnv("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Kafka: KafkaConfig{
			Brokers: strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
			Topic:   getEnv("KAFKA_TOPIC", "company-events"),
			Enabled: getBoolEnv("KAFKA_ENABLED", true),
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", "your-256-bit-secret-key-here"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
