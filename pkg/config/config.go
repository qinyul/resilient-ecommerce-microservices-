package config

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv   string
	Port     string
	Database DatabaseConfig
	RabbitMQ RabbitMQConfig
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type RabbitMQConfig struct {
	User            string
	Password        string
	Host            string
	Port            string
	ExchangeName    string
	OrderCreatedKey string
}

// Load loads configuration from .env (if present) and environment variables.
func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		slog.Warn("No .env file found, relying on environment variables", "error", err)
	}

	return &Config{
		AppEnv: getEnv("APP_ENV", "development"),
		Port:   getEnv("ORDER_PORT", "50051"),
		Database: DatabaseConfig{
			Host:     getEnv("ORDER_DB_HOST", "localhost"),
			Port:     getEnv("ORDER_DB_PORT", "5432"),
			User:     getEnv("ORDER_DB_USER", "postgres"),
			Password: getEnv("ORDER_DB_PASSWORD", "postgres"),
			Name:     getEnv("ORDER_DB_NAME", "orders"),
			SSLMode:  getEnv("ORDER_DB_SSLMODE", "disable"),
		},
		RabbitMQ: RabbitMQConfig{
			User:            getEnv("RABBITMQ_DEFAULT_USER", "guest"),
			Password:        getEnv("RABBITMQ_DEFAULT_PASS", "guest"),
			Host:            getEnv("RABBITMQ_HOST", "localhost"),
			Port:            getEnv("RABBITMQ_PORT", "5672"),
			ExchangeName:    getEnv("RABBITMQ_EXCHANGE", "order_exchange"),
			OrderCreatedKey: getEnv("RABBITMQ_ROUTING_KEY", "order_created"),
		},
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
