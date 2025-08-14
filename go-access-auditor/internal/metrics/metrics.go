package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	EventsIngested = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name:"access_auditor_events_ingested_total", Help:"Eventos recebidos"},
		[]string{"source"},
	)
	SensitiveMatches = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name:"access_auditor_sensitive_matches_total", Help:"Comandos sensíveis detectados"},
		[]string{"rule"},
	)
)

func MustRegister() {
	prometheus.MustRegister(EventsIngested, SensitiveMatches)
}
func Handler() http.Handler { return promhttp.Handler() }