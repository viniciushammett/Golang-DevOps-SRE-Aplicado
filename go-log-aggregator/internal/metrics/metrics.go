package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	LogsIngested = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "logagg_logs_ingested_total",
			Help: "Total de linhas de log ingeridas",
		}, []string{"source"},
	)
	IngestErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "logagg_ingest_errors_total",
			Help: "Erros de ingestão por origem",
		}, []string{"source"},
	)
	IngestLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "logagg_ingest_latency_seconds",
			Help: "Latência por linha no pipeline de ingestão",
			Buckets: prometheus.DefBuckets,
		}, []string{"source"},
	)
	APIQueries = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "logagg_api_queries_total",
			Help: "Consultas realizadas na API de busca",
		}, []string{"route"},
	)
)

func MustRegister() {
	prometheus.MustRegister(LogsIngested, IngestErrors, IngestLatency, APIQueries)
}

func Handler() http.Handler { return promhttp.Handler() }