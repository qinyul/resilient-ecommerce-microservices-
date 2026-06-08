package config

import (
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv   string
	Port     string
	Database DatabaseConfig
	RabbitMQ RabbitMQConfig
	Product   ServiceConfig
	Gateway   GatewayConfig
	Telemetry TelemetryConfig
}

type ServiceConfig struct {
	Port     string
	Database DatabaseConfig
}

type TelemetryConfig struct {
	OTLPEndpoint string
	SampleRate   float64
}

type GatewayConfig struct {
	Port               string
	OrderServiceAddr   string
	ProductServiceAddr string
	RateLimitRate      float64
	RateLimitCap       float64
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
		Product: ServiceConfig{
			Port: getEnv("PRODUCT_PORT", "50052"),
			Database: DatabaseConfig{
				Host:     getEnv("PRODUCT_DB_HOST", "localhost"),
				Port:     getEnv("PRODUCT_DB_PORT", "5432"),
				User:     getEnv("PRODUCT_DB_USER", "postgres"),
				Password: getEnv("PRODUCT_DB_PASSWORD", "postgres"),
				Name:     getEnv("PRODUCT_DB_NAME", "products"),
				SSLMode:  getEnv("PRODUCT_DB_SSLMODE", "disable"),
			},
		},
		Gateway: GatewayConfig{
			Port:               getEnv("GATEWAY_PORT", "8080"),
			OrderServiceAddr:   getEnv("ORDER_SERVICE_ADDR", "localhost:50051"),
			ProductServiceAddr: getEnv("PRODUCT_SERVICE_ADDR", "localhost:50052"),
			RateLimitRate:      5.0,  // 5 requests per second
			RateLimitCap:       10.0, // Burst up to 10
		},
		Telemetry: TelemetryConfig{
			OTLPEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "otel-collector:4317"),
			SampleRate:   getEnvFloat("OTEL_SAMPLING_RATE", 1.0),
		},
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value, exists := os.LookupEnv(key); exists {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}
