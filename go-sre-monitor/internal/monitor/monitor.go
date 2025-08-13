package monitor

import (
	"context"
	"net/http"
	"time"

	"github.com/viniciushammett/go-sre-monitor/internal/config"
	"github.com/viniciushammett/go-sre-monitor/internal/logger"
	"github.com/viniciushammett/go-sre-monitor/internal/metrics"
)

type Monitor struct {
	log      *logger.Logger
	cfg      *config.Config
	interval time.Duration
	client   *http.Client
}

func New(log *logger.Logger, cfg *config.Config, interval time.Duration) *Monitor {
	return &Monitor{
		log:      log,
		cfg:      cfg,
		interval: interval,
		client:   &http.Client{},
	}
}

func (m *Monitor) Run(ctx context.Context) {
	t := time.NewTicker(m.interval)
	defer t.Stop()

	m.log.Info().Int("services", len(m.cfg.Services)).Msg("monitor loop started")

	// Primeiro disparo imediato
	m.probeAll(ctx)

	for {
		select {
		case <-ctx.Done():
			m.log.Info().Msg("monitor loop stopped")
			return
		case <-t.C:
			m.probeAll(ctx)
		}
	}
}

func (m *Monitor) probeAll(ctx context.Context) {
	for _, svc := range m.cfg.Services {
		svc := svc
		go m.probeOne(ctx, svc)
	}
}

func (m *Monitor) probeOne(ctx context.Context, svc config.Service) {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, svc.Method, svc.URL, nil)
	if err != nil {
		m.fail(svc, "build_request_error")
		return
	}
	for k, v := range svc.Headers {
		req.Header.Set(k, v)
	}

	// Timeout por serviço
	c := &http.Client{Timeout: svc.Timeout}
	resp, err := c.Do(req)
	if err != nil {
		m.fail(svc, "request_error")
		return
	}
	defer resp.Body.Close()

	elapsed := time.Since(start)
	code := resp.StatusCode

	metrics.ProbeDuration.WithLabelValues(svc.Name, httpStatusLabel(code)).Observe(elapsed.Seconds())

	if code != svc.ExpectedStatus {
		m.log.Warn().
			Str("service", svc.Name).
			Str("url", svc.URL).
			Int("got", code).
			Int("want", svc.ExpectedStatus).
			Msg("status mismatch")
		metrics.ProbeUp.WithLabelValues(svc.Name).Set(0)
		metrics.ProbeFailures.WithLabelValues(svc.Name).Inc()
		return
	}

	// Latência/SLO
	if svc.SLOLatency > 0 && elapsed > svc.SLOLatency {
		m.log.Warn().
			Str("service", svc.Name).
			Dur("elapsed", elapsed).
			Dur("slo", svc.SLOLatency).
			Msg("latency SLO breach")
		metrics.SLOBreaches.WithLabelValues(svc.Name).Inc()
	}

	metrics.ProbeUp.WithLabelValues(svc.Name).Set(1)
	m.log.Debug().Str("service", svc.Name).Dur("elapsed", elapsed).Int("status", code).Msg("ok")
}

func (m *Monitor) fail(svc config.Service, reason string) {
	metrics.ProbeUp.WithLabelValues(svc.Name).Set(0)
	metrics.ProbeFailures.WithLabelValues(svc.Name).Inc()
	m.log.Error().Str("service", svc.Name).Str("url", svc.URL).Str("reason", reason).Msg("probe failed")
}

func httpStatusLabel(code int) string {
	return http.StatusText(code)
}