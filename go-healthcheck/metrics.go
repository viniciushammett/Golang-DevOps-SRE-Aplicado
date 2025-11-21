package main

import (
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	healthStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "healthchecker_up",
			Help: "1 se o alvo está UP, 0 se está DOWN",
		},
		[]string{"url"},
	)

	healthLatency = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "healthchecker_latency_ms",
			Help: "Latência do último healthcheck em milissegundos",
		},
		[]string{"url"},
	)
)

func init() {
	prometheus.MustRegister(healthStatus, healthLatency)
}

// startMetricsServer roda um loop de healthcheck e expõe métricas em /metrics
func startMetricsServer(url string, timeoutSec int, interval time.Duration, listenAddr string) {
	// Loop que atualiza métricas periodicamente
	go func() {
		for {
			res, _ := checkHTTP(url, timeoutSec)
			if res != nil {
				if res.Status == "UP" {
					healthStatus.WithLabelValues(res.URL).Set(1)
				} else {
					healthStatus.WithLabelValues(res.URL).Set(0)
				}
				healthLatency.WithLabelValues(res.URL).Set(float64(res.ElapsedMS))
			}
			time.Sleep(interval)
		}
	}()

	http.Handle("/metrics", promhttp.Handler())

	log.Printf("💡 Servindo métricas em http://%s/metrics (checando %s a cada %s)\n",
		listenAddr, url, interval)

	if err := http.ListenAndServe(listenAddr, nil); err != nil {
		log.Fatal("Erro ao iniciar servidor de métricas:", err)
	}
}
