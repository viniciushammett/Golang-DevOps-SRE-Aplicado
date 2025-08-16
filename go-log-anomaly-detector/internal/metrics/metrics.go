package metrics

import (
	"net/http"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	LogsIngested = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name:"lad_logs_ingested_total", Help:"Logs recebidos"},
		[]string{"source"},
	)
	Anomalies = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name:"lad_anomalies_total", Help:"Anomalias detectadas"},
		[]string{"rule","kind"},
	)
	WindowGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name:"lad_window_count", Help:"Eventos na janela"},
		[]string{"rule"},
	)
)

func MustRegister() {
	prometheus.MustRegister(LogsIngested, Anomalies, WindowGauge)
}
func Handler() http.Handler { return promhttp.Handler() }