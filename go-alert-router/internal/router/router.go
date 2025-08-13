package router

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/viniciushammett/go-alert-router/internal/config"
	"github.com/viniciushammett/go-alert-router/internal/logger"
	"github.com/viniciushammett/go-alert-router/internal/metrics"
	"github.com/viniciushammett/go-alert-router/internal/model"
	"github.com/viniciushammett/go-alert-router/internal/notify"
	"github.com/viniciushammett/go-alert-router/internal/store"
)

type Router struct {
	log    *logger.Logger
	store  *store.Store
	cfg    *config.Config
	slack  *notify.Slack
	email  *notify.Email

	queues map[string]chan model.Alert // por rota (para agrupamento)
}

func New(log *logger.Logger, st *store.Store, cfg *config.Config, slack *notify.Slack, email *notify.Email) *Router {
	return &Router{
		log: log, store: st, cfg: cfg, slack: slack, email: email,
		queues: make(map[string]chan model.Alert),
	}
}

func (r *Router) Start(ctx context.Context) {
	// um worker por rota para agrupar e entregar em batch pela janela
	for _, rt := range r.cfg.Routes {
		ch := make(chan model.Alert, 1024)
		r.queues[rt.Name] = ch
		go r.batchWorker(ctx, rt, ch)
	}
}

func (r *Router) Ingest(alerts []model.Alert, source string) {
	metrics.AlertsIngested.WithLabelValues(source).Add(float64(len(alerts)))
	for _, a := range alerts {
		a.EnsureFingerprint()
		if r.isSilenced(a) {
			metrics.AlertsDropped.WithLabelValues("silenced").Inc()
			continue
		}
		// roteamento por matchers
		for _, rt := range r.cfg.Routes {
			if matchAll(rt.Matchers, a.Labels) {
				// rate limit
				ok, _ := r.store.IncRate(rt.Name, rt.RateLimitPerMin)
				if !ok {
					metrics.AlertsDropped.WithLabelValues("ratelimit").Inc()
					continue
				}
				// dedupe
				seen, _ := r.store.SeenRecently(a.Fingerprint, rt.DedupeWindow)
				if seen {
					metrics.AlertsDropped.WithLabelValues("dedupe").Inc()
					continue
				}
				_ = r.store.MarkSeen(a.Fingerprint)
				// enqueue
				if q, ok := r.queues[rt.Name]; ok {
					select { case q <- a: default:
						metrics.AlertsDropped.WithLabelValues("queue_full").Inc()
					}
					metrics.QueueDepth.WithLabelValues(rt.Name).Set(float64(len(q)))
				}
			}
		}
	}
}

func (r *Router) batchWorker(ctx context.Context, rt config.Route, ch <-chan model.Alert) {
	t := time.NewTicker(max(rt.GroupWindow, 5*time.Second))
	defer t.Stop()
	var batch []model.Alert

	flush := func() {
		if len(batch) == 0 { return }
		payload := formatBatch(batch)
		dest := ""
		var err error
		if rt.Slack != nil && rt.Slack.Webhook != "" {
			dest = "slack"
			err = r.slack.Send(rt.Slack.Webhook, payload)
		}
		if err == nil && len(rt.EmailTo) > 0 {
			dest = "email"
			err = r.email.Send(rt.EmailTo, "[alert-router] "+shortTitle(batch), payload)
		}
		if err != nil {
			metrics.DeliveryErrors.WithLabelValues(dest).Inc()
			_ = r.store.PutDLQ(store.DLQItem{
				When: time.Now(), Route: rt.Name, Dest: dest, Alert: batch, Error: err.Error(),
			})
			// retry simples com backoff
			time.Sleep(2 * time.Second)
		} else {
			metrics.Deliveries.WithLabelValues(dest).Add(1)
		}
		batch = batch[:0]
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case a := <-ch:
			batch = append(batch, a)
			metrics.QueueDepth.WithLabelValues(rt.Name).Set(float64(len(ch)))
		case <-t.C:
			flush()
		}
	}
}

func (r *Router) isSilenced(a model.Alert) bool {
	sils, _ := r.store.ListSilences()
	for _, s := range sils {
		if time.Now().After(s.Until) { continue }
		if val, ok := a.Labels[s.Label]; ok {
			if ok, _ := regexp.MatchString(s.Regex, val); ok { return true }
		}
	}
	return false
}

func matchAll(ms []config.Matcher, labels map[string]string) bool {
	for _, m := range ms {
		v, ok := labels[m.Label]; if !ok { return false }
		ok2, _ := regexp.MatchString(m.Regex, v)
		if !ok2 { return false }
	}
	return true
}

func formatBatch(alerts []model.Alert) string {
	var b strings.Builder
	title := shortTitle(alerts)
	b.WriteString("*" + title + "*\n")
	for _, a := range alerts {
		b.WriteString("- ")
		if s, ok := a.Labels["summary"]; ok {
			b.WriteString(s)
		} else if m, ok := a.Annotations["summary"]; ok {
			b.WriteString(m)
		} else {
			b.WriteString(a.Fingerprint[:8])
		}
		if sev, ok := a.Labels["severity"]; ok {
			b.WriteString(" [sev:" + sev + "]")
		}
		if inst, ok := a.Labels["instance"]; ok {
			b.WriteString(" (" + inst + ")")
		}
		b.WriteString("\n")
	}
	return b.String()
}
func shortTitle(alerts []model.Alert) string {
	if len(alerts) == 0 { return "alerts" }
	if sev, ok := alerts[0].Labels["severity"]; ok {
		return fmt.Sprintf("%d alert(s) severity=%s", len(alerts), sev)
	}
	return fmt.Sprintf("%d alert(s)", len(alerts))
}
func max(d time.Duration, min time.Duration) time.Duration { if d < min { return min }; return d }