package telemetry

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const (
	serviceName = "p5"
)

var version = "dev"

type Telemetry struct {
	tracerProvider *sdktrace.TracerProvider
	loggerProvider *sdklog.LoggerProvider
	Logger         *slog.Logger
}

func SetVersion(v string) {
	version = v
}

type Options struct {
	Debug bool
}

func Setup(ctx context.Context, opts Options) (*Telemetry, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		return newNoopTelemetry(opts.Debug), nil
	}

	res, err := newResource(ctx)
	if err != nil {
		return nil, err
	}

	tracerProvider, err := newTracerProvider(ctx, res)
	if err != nil {
		return nil, err
	}

	loggerProvider, err := newLoggerProvider(ctx, res)
	if err != nil {
		_ = tracerProvider.Shutdown(ctx)
		return nil, err
	}

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	global.SetLoggerProvider(loggerProvider)

	logger := otelslog.NewLogger(serviceName,
		otelslog.WithLoggerProvider(loggerProvider),
	)

	return &Telemetry{
		tracerProvider: tracerProvider,
		loggerProvider: loggerProvider,
		Logger:         logger,
	}, nil
}

func (t *Telemetry) Shutdown(ctx context.Context) error {
	if t.tracerProvider == nil && t.loggerProvider == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var errs []error

	if t.loggerProvider != nil {
		if err := t.loggerProvider.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if t.tracerProvider != nil {
		if err := t.tracerProvider.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func newResource(ctx context.Context) (*resource.Resource, error) {
	return resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(version),
		),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
	)
}

func newTracerProvider(ctx context.Context, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, err
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
		),
	), nil
}

func newLoggerProvider(ctx context.Context, res *resource.Resource) (*sdklog.LoggerProvider, error) {
	exporter, err := otlploggrpc.New(ctx)
	if err != nil {
		return nil, err
	}

	return sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	), nil
}

func newNoopTelemetry(debug bool) *Telemetry {
	var handler slog.Handler
	if debug {
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	} else {
		handler = slog.NewTextHandler(discard{}, nil)
	}
	return &Telemetry{
		Logger: slog.New(handler),
	}
}

func NewNoop() *Telemetry {
	return newNoopTelemetry(false)
}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }
