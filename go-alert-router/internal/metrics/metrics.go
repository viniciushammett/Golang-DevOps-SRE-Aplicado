package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	AlertsIngested = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "alert_router_ingested_total", Help: "Alertas recebidos"},
		[]string{"source"},
	)
	AlertsDropped = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "alert_router_dropped_total", Help: "Alertas descartados (silenced/dedupe/ratelimit)"},
		[]string{"reason"},
	)
	Deliveries = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "alert_router_deliveries_total", Help: "Entregas por destino"},
		[]string{"dest"},
	)
	DeliveryErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "alert_router_delivery_errors_total", Help: "Erros de entrega"},
		[]string{"dest"},
	)
	QueueDepth = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: "alert_router_queue_depth", Help: "Tamanho da fila por rota"},
		[]string{"route"},
	)
	OpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "alert_router_operation_duration_seconds", Help: "Duração de operações"},
		[]string{"op"},
	)
)

func MustRegister() {
	prometheus.MustRegister(AlertsIngested, AlertsDropped, Deliveries, DeliveryErrors, QueueDepth, OpDuration)
}
func Handler() http.Handler { return promhttp.Handler() }