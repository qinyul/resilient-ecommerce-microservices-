package telemetry

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// InitTracerProvider initializes an OTLP exporter, and configures the corresponding trace and
// metric providers. It returns a function to cleanly shutdown the provider.
func InitTracerProvider(ctx context.Context, serviceName, environment, endpoint string) (func(context.Context) error, error) {
	// Configure OTLP Exporter via gRPC
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// 3. Define the resource (service name, version, environment)
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.DeploymentEnvironment(environment),
		),
	)	
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// 4. Create TraceProvider with a BatchSpanProcessor
	// Use 1.0 (100%) for development, but dial this down (e.g., 0.05) in production
	sampleRate := 1.0
	if rateStr := os.Getenv("OTEL_SAMPLING_RATE"); rateStr != "" {
		if parsedRate, err := strconv.ParseFloat(rateStr, 64); err == nil {
			sampleRate = parsedRate
		}
	}

	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(sampleRate))

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5 * time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
		sdktrace.WithResource(res),
	)

	// 5. Register our TraceProvider as the global so any imported
	// instrumentation in the future will default to using it
	otel.SetTracerProvider(tp)

	// 6. Register W3C Trace Context and Baggage as global propagators
	// This ensures traces propage accross HTTP/gRPC/AMQP boundaries seamlessly.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}
