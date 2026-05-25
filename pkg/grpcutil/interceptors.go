package grpcutil

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
)

// NewServer creates a new gRPC server with recovery, logging, and metrics interceptors.
func NewServer(logger *slog.Logger) *grpc.Server {
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			recovery.UnaryServerInterceptor(),
			logging.UnaryServerInterceptor(InterceptorLogger(logger)),
			grpc_prometheus.UnaryServerInterceptor,
		),
	)
	
	// Register prometheus metrics on the gRPC server
	grpc_prometheus.Register(srv)
	
	return srv
}

// StartMetricsServer starts an HTTP server for Prometheus metrics.
func StartMetricsServer(port string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: mux,
	}

	go func() {
		slog.Info("Metrics server starting", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("metrics server failed", "error", err)
		}
	}()

	return srv
}

// InterceptorLogger adapts slog logger to interceptor logger.
func InterceptorLogger(l *slog.Logger) logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		var slogLvl slog.Level
		switch lvl {
		case logging.LevelDebug:
			slogLvl = slog.LevelDebug
		case logging.LevelInfo:
			slogLvl = slog.LevelInfo
		case logging.LevelWarn:
			slogLvl = slog.LevelWarn
		case logging.LevelError:
			slogLvl = slog.LevelError
		default:
			slogLvl = slog.LevelInfo
		}
		l.Log(ctx, slogLvl, msg, fields...)
	})
}
