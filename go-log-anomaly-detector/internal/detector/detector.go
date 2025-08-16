package detector

import (
	"crypto/rand"
	"encoding/hex"
	"time"
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/viniciushammett/go-log-anomaly-detector/internal/logger"
	"github.com/viniciushammett/go-log-anomaly-detector/internal/metrics"
	"github.com/viniciushammett/go-log-anomaly-detector/internal/notify"
	"github.com/viniciushammett/go-log-anomaly-detector/internal/rules"
	"github.com/viniciushammett/go-log-anomaly-detector/internal/store"
	"github.com/viniciushammett/go-log-anomaly-detector/internal/util"
	"github.com/viniciushammett/go-log-anomaly-detector/internal/ml"
)

var tracer = otel.Tracer("detector")

type Detector struct {
	log   *logger.Logger
	db    *store.Store
	rs    *rules.Set
	slack *notify.Slack
	ml    *ml.Detector

	win map[string]*util.Sliding // por regra
}

func New(log *logger.Logger, db *store.Store, rs *rules.Set, slack *notify.Slack, mlDet *ml.Detector) *Detector {
	return &Detector{log: log, db: db, rs: rs, slack: slack, ml: mlDet, win: map[string]*util.Sliding{}}
}

type LogLine struct {
	TS     time.Time
	Source string
	Msg    string
	Meta   map[string]string
}

func (d *Detector) Submit(l LogLine) {
	ctx, span := tracer.Start(context.Background(), "Submit")
	defer span.End()
	if l.TS.IsZero() { l.TS = time.Now() }
	metrics.LogsIngested.WithLabelValues(l.Source).Inc()
	span.SetAttributes(
		attribute.String("source", l.Source),
		attribute.Int64("ts_unix", l.TS.Unix()),
	)

	// ML (auto-anomalia independente de regras)
	if d.ml != nil {
		if anom, score, cnt, key := d.ml.Observe(l); anom {
			d.raise(ctx, "ml", rules.Rule{Name:"ml-auto"}, cnt, l, score)
		}
	}

	// Regras baseadas em regex + janela
	for _, r := range d.rs.Items {
		if r.RE.MatchString(l.Msg) {
			s := d.win[r.Name]
			if s == nil { s = &util.Sliding{}; d.win[r.Name] = s }
			s.Add(util.Point{TS:l.TS, Data:l.Msg}, r.Window)
			metrics.WindowGauge.WithLabelValues(r.Name).Set(float64(s.Count()))
			if r.MinCount > 0 && s.Count() >= r.MinCount {
				d.raise(ctx, "threshold", r, s.Count(), l, 0)
				d.win[r.Name] = &util.Sliding{} // reset anti-rajada
			}
		}
	}
}

func (d *Detector) raise(ctx context.Context, kind string, r rules.Rule, count int, l LogLine, score float64) {
	_, span := tracer.Start(ctx, "Raise")
	defer span.End()
	span.SetAttributes(
		attribute.String("rule", r.Name),
		attribute.String("kind", kind),
		attribute.Int("count", count),
		attribute.Float64("score", score),
	)

	id := randID()
	a := store.Anomaly{
		ID: id, When: time.Now(), Rule: r.Name, Kind: kind, Sample: l.Msg,
		Count: count, Window: r.Window.String(), Severity: r.Severity, Meta: l.Meta,
	}
	_ = d.db.PutAnomaly(a)
	metrics.Anomalies.WithLabelValues(r.Name, kind).Inc()
	_ = d.slack.Send(notify.Format(r.Name, kind, count, r.Window.String(), l.Msg, r.Severity))
	d.log.Warn().Str("rule", r.Name).Str("kind", kind).Int("count", count).Msg("anomaly")
}

func randID() string { b:=make([]byte,6); _,_ = rand.Read(b); return hex.EncodeToString(b) }
