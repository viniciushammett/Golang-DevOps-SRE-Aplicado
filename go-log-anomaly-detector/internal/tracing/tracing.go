package tracing

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"google.golang.org/grpc"
)

type Config struct {
	Enabled     bool
	ServiceName string
	OTLPEndpoint string
	SampleRatio float64
}

type Closer func(context.Context) error

func Init(ctx context.Context, cfg Config) (Closer, error) {
	if !cfg.Enabled {
		return func(context.Context) error { return nil }, nil
	}
	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
		otlptracegrpc.WithDialOption(grpc.WithBlock()),
	)
	if err != nil { return nil, err }

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRatio))),
		sdktrace.WithBatcher(exp, sdktrace.WithMaxExportBatchSize(512), sdktrace.WithBatchTimeout(1*time.Second)),
		sdktrace.WithResource(resource.NewSchemaless(
			semconv.ServiceName(cfg.ServiceName),
		)),
	)
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}