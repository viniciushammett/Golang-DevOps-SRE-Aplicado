package metrics

import (
	"net/http"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	DeploysStarted = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name:"do_deploys_started_total",Help:"Deploys iniciados"},
		[]string{"app","strategy"},
	)
	DeploysSucceeded = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name:"do_deploys_succeeded_total",Help:"Deploys concluídos com sucesso"},
		[]string{"app","strategy"},
	)
	DeploysFailed = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name:"do_deploys_failed_total",Help:"Deploys que falharam (rollback)"},
		[]string{"app","strategy","reason"},
	)
	StepDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name:"do_step_duration_seconds",Help:"Duração por etapa do deploy"},
		[]string{"app","strategy","step"},
	)
)

func MustRegister() {
	prometheus.MustRegister(DeploysStarted, DeploysSucceeded, DeploysFailed, StepDuration)
}
func Handler() http.Handler { return promhttp.Handler() }