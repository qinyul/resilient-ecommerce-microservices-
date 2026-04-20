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

	"github.com/qinyul/resilient-ecommerce-microservices/internal/order/broker"
	"github.com/qinyul/resilient-ecommerce-microservices/internal/order/client"
	"github.com/qinyul/resilient-ecommerce-microservices/internal/order/handler"
	"github.com/qinyul/resilient-ecommerce-microservices/internal/order/repository"
	"github.com/qinyul/resilient-ecommerce-microservices/internal/order/service"
	"github.com/qinyul/resilient-ecommerce-microservices/internal/pkg/db"
	pb "github.com/qinyul/resilient-ecommerce-microservices/pb/order/v1"
	pkgbroker "github.com/qinyul/resilient-ecommerce-microservices/pkg/broker"
	"github.com/qinyul/resilient-ecommerce-microservices/pkg/config"
	"github.com/qinyul/resilient-ecommerce-microservices/pkg/rabbitmq"
	"github.com/qinyul/resilient-ecommerce-microservices/worker"
	"google.golang.org/grpc"
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
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.Name,
		SSLMode:  cfg.Database.SSLMode,
	})
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	// Initialize RabbitMQ Client
	rabbitMQURL := fmt.Sprintf("amqp://%s:%s@%s:%s/",
		cfg.RabbitMQ.User, cfg.RabbitMQ.Password, cfg.RabbitMQ.Host, cfg.RabbitMQ.Port)
	rmqClient := rabbitmq.NewClient(rabbitMQURL)
	defer rmqClient.Close()

	// Wait for RabbitMQ to connect (optional, depends on how resilient you want the start to be)
	// For now, let's just log and continue, the client handles reconnections
	slog.Info("Initializing RabbitMQ client...")

	// Wire dependencies
	repo := repository.NewPostgresOrderRepository(database)
	messageBroker := broker.NewRabbitMQBroker(rmqClient)
	productClient := client.NewProductStubClient()
	orderService := service.NewOrderService(repo, messageBroker, productClient)
	orderHandler := handler.NewOrderHandler(orderService)

	// Setup Payment Completed Consumer
	paymentHandler := handler.NewPaymentCompletedHandler(orderService)
	paymentConsumer := pkgbroker.NewEventConsumer(
		rmqClient,
		"order_payment_completed",
		"order_exchange",
		"payment_completed",
		"order_service",
		paymentHandler.HandlePaymentCompleted,
	)
	go paymentConsumer.Start(ctx)

	// Initialize and Start the Background Workder
	relayWorkder := worker.NewOutboxRelayWorker(database, messageBroker)
	go relayWorkder.Start(ctx)

	// Start gRPC server
	addr := fmt.Sprintf(":%s", cfg.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		slog.Error("failed to listen", "error", err, "port", cfg.Port)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterOrderServiceServer(grpcServer, orderHandler)

	if cfg.AppEnv != "production" {
		reflection.Register(grpcServer)
	}

	serverError := make(chan error, 1)
	go func() {
		slog.Info("Order Service is starting", "addr", addr, "env", cfg.AppEnv)
		if err := grpcServer.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			slog.Error("failed to serve", "error", err)
			serverError <- err
		}
	}()

	select {
	case err := <-serverError:
		slog.Error("gRPC server crashed", "error", err)
	case <-ctx.Done():
		slog.Info("shutting down Order Service", "signal", "SIGINT/SIGTERM")
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
		slog.Info("Order Service stopped gracefully")
	}

}
