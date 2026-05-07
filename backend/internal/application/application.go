package application

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"restaurantops/backend/internal/config"
	"restaurantops/backend/internal/events"
	"restaurantops/backend/internal/httpapi"
)

// Application owns process-scoped dependencies and their lifecycle.
type Application struct {
	database  *pgxpool.Pool
	writer    *kafka.Writer
	publisher *events.OutboxPublisher
	http      *http.Server
	traces    *sdktrace.TracerProvider
}

func New(ctx context.Context, settings config.Settings) (*Application, error) {
	database, err := pgxpool.New(ctx, settings.DatabaseURL)
	if err != nil {
		return nil, err
	}
	if err := database.Ping(ctx); err != nil {
		database.Close()
		return nil, err
	}
	app := &Application{database: database}
	app.writer = &kafka.Writer{
		Addr:                   kafka.TCP(settings.KafkaBrokers...),
		Topic:                  "restaurantops.events",
		Balancer:               &kafka.Hash{},
		RequiredAcks:           kafka.RequireAll,
		AllowAutoTopicCreation: true,
		WriteTimeout:           10 * time.Second,
	}
	app.publisher = events.NewOutboxPublisher(database, app.writer)
	if settings.OTelEndpoint != "" {
		exporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpoint(settings.OTelEndpoint), otlptracehttp.WithInsecure())
		if err != nil {
			app.Close()
			return nil, err
		}
		app.traces = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(resource.NewWithAttributes("", attribute.String("service.name", "restaurantops-backend"))),
		)
		otel.SetTracerProvider(app.traces)
	}
	app.http = &http.Server{
		Addr: ":8080", Handler: httpapi.NewRouter(database, settings),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second,
	}
	return app, nil
}

func (app *Application) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go app.publisher.Run(ctx)
	failures := make(chan error, 1)
	go func() { failures <- app.http.ListenAndServe() }()
	select {
	case <-ctx.Done():
		shutdown, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		return app.http.Shutdown(shutdown)
	case err := <-failures:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func (app *Application) Close() {
	if app.writer != nil {
		_ = app.writer.Close()
	}
	if app.traces != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = app.traces.Shutdown(ctx)
	}
	app.database.Close()
}
