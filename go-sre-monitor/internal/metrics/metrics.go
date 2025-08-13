package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	ProbeDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "probe_duration_seconds",
			Help:    "HTTP probe duration per service",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "code"},
	)

	ProbeUp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "probe_up",
			Help: "Service up (1) or down (0)",
		},
		[]string{"service"},
	)

	ProbeFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "probe_failures_total",
			Help: "Total probe failures per service",
		},
		[]string{"service"},
	)

	SLOBreaches = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "slo_latency_breaches_total",
			Help: "Total SLO latency breaches",
		},
		[]string{"service"},
	)
)

func MustRegister() {
	prometheus.MustRegister(ProbeDuration, ProbeUp, ProbeFailures, SLOBreaches)
}