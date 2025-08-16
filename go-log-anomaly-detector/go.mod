module github.com/viniciushammett/go-log-anomaly-detector

go 1.22

require (
	github.com/go-chi/chi/v5 v5.1.0
	github.com/prometheus/client_golang v1.19.1
	github.com/rs/zerolog v1.33.0
	go.etcd.io/bbolt v1.3.9
	gopkg.in/yaml.v3 v3.0.1
	go.opentelemetry.io/otel v1.24.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.24.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.24.0
	go.opentelemetry.io/otel/sdk v1.24.0
	go.opentelemetry.io/otel/semconv/v1.21.0 v1.21.0
)