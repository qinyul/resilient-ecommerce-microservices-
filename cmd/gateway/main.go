package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/qinyul/resilient-ecommerce-microservices/internal/gateway/handler"
	"github.com/qinyul/resilient-ecommerce-microservices/internal/gateway/middleware"
	orderpb "github.com/qinyul/resilient-ecommerce-microservices/pb/order/v1"
	productpb "github.com/qinyul/resilient-ecommerce-microservices/pb/product/v1"
	"github.com/qinyul/resilient-ecommerce-microservices/pkg/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Initialize structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	slog.Info("Starting API Gateway...",
		"port", cfg.Gateway.Port,
		"order_service", cfg.Gateway.OrderServiceAddr,
		"product_service", cfg.Gateway.ProductServiceAddr,
	)

	// Set up gRPC client connections
	orderConn, err := grpc.NewClient(cfg.Gateway.OrderServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("failed to connect to order service", "error", err)
		os.Exit(1)
	}
	defer orderConn.Close()

	productConn, err := grpc.NewClient(cfg.Gateway.ProductServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("failed to connect to product service", "error", err)
		os.Exit(1)
	}
	defer productConn.Close()

	orderClient := orderpb.NewOrderServiceClient(orderConn)
	productClient := productpb.NewProductServiceClient(productConn)

	limiter := middleware.NewRateLimiter(cfg.Gateway.RateLimitRate, cfg.Gateway.RateLimitCap)

	// Define Router using Go 1.22+ pattern matching in ServeMux
	mux := http.NewServeMux()

	// Apply Middleware manually to handlers
	mux.HandleFunc("POST /api/v1/orders", middleware.RateLimit(limiter, middleware.Logging(handler.HandleCreateOrder(orderClient))))
	mux.HandleFunc("GET /api/v1/orders/{id}", middleware.RateLimit(limiter, middleware.Logging(handler.HandleGetOrder(orderClient))))
	mux.HandleFunc("GET /api/v1/orders", middleware.RateLimit(limiter, middleware.Logging(handler.HandleListOrders(orderClient))))
	mux.HandleFunc("POST /api/v1/products", middleware.RateLimit(limiter, middleware.Logging(handler.HandleCreateProduct(productClient))))
	mux.HandleFunc("GET /api/v1/products/{id}", middleware.RateLimit(limiter, middleware.Logging(handler.HandleGetProduct(productClient))))
	mux.HandleFunc("GET /healthz", handler.HandleHealthz)

	server := &http.Server{
		Addr:    ":" + cfg.Gateway.Port,
		Handler: mux,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("Gateway HTTP server is running", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("failed to serve HTTP gateway", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down HTTP gateway gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("failed to shutdown HTTP gateway", "error", err)
		server.Close()
	}
	slog.Info("HTTP gateway stopped gracefully")
}
