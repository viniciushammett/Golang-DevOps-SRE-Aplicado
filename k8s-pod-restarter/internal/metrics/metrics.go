package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	RestartsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pod_restarter_restarts_total",
			Help: "Total restarts (pods deleted) grouped by namespace and reason",
		}, []string{"namespace", "reason"},
	)

	ErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pod_restarter_errors_total",
			Help: "Errors while attempting restarts",
		}, []string{"namespace", "reason"},
	)

	LastRestart = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "pod_restarter_last_restart_timestamp",
			Help: "Unix timestamp of last restart per namespace",
		}, []string{"namespace"},
	)

	OpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pod_restarter_operation_duration_seconds",
			Help:    "Duration of restart operations",
			Buckets: prometheus.DefBuckets,
		}, []string{"namespace"},
	)
)

func MustRegister() {
	prometheus.MustRegister(RestartsTotal, ErrorsTotal, LastRestart, OpDuration)
}

func Handler() http.Handler { return promhttp.Handler() }