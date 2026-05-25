package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/qinyul/resilient-ecommerce-microservices/internal/pkg/db"
	"github.com/qinyul/resilient-ecommerce-microservices/internal/product/handler"
	"github.com/qinyul/resilient-ecommerce-microservices/internal/product/repository"
	"github.com/qinyul/resilient-ecommerce-microservices/internal/product/service"
	pb "github.com/qinyul/resilient-ecommerce-microservices/pb/product/v1"
	"github.com/qinyul/resilient-ecommerce-microservices/pkg/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Initialize structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Initialize Database
	database, err := db.NewPostgresDB(ctx, db.Config{
		Host:     cfg.Product.Database.Host,
		Port:     cfg.Product.Database.Port,
		User:     cfg.Product.Database.User,
		Password: cfg.Product.Database.Password,
		DBName:   cfg.Product.Database.Name,
		SSLMode:  cfg.Product.Database.SSLMode,
	})
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	// Wire dependencies
	repo := repository.NewPostgresProductRepository(database)
	productService := service.NewProductService(repo)
	productHandler := handler.NewProductHandler(productService)

	// Start gRPC server
	addr := fmt.Sprintf(":%s", cfg.Product.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		slog.Error("failed to listen", "error", err, "port", cfg.Product.Port)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			// 1. Recover from panics and return codes.Internal
			recovery.UnaryServerInterceptor(),
			// 2. Add your structured request logging here
			// 3. Add metrics (Prometheus/StatsD)
		),
	)

	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("product.v1.ProductService", healthpb.HealthCheckResponse_SERVING)
	pb.RegisterProductServiceServer(grpcServer, productHandler)

	if cfg.AppEnv != "production" {
		reflection.Register(grpcServer)
	}

	serverError := make(chan error, 1)
	go func() {
		slog.Info("Product Service is starting", "addr", addr, "env", cfg.AppEnv)
		if err := grpcServer.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			slog.Error("failed to serve", "error", err)
			serverError <- err
		}
	}()

	select {
	case err := <-serverError:
		slog.Error("gRPC server crashed", "error", err)
	case <-ctx.Done():
		slog.Info("shutting down Product Service", "signal", "SIGINT/SIGTERM")
	}

	// Context for graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-shutdownCtx.Done():
		slog.Warn("Graceful shutdown timed out, forcing server stop")
		grpcServer.Stop()
	case <-stopped:
		slog.Info("Product Service stopped gracefully")
	}
}
