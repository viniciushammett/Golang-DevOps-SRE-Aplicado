package detector

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/viniciushammett/go-log-anomaly-detector/internal/logger"
	"github.com/viniciushammett/go-log-anomaly-detector/internal/metrics"
	"github.com/viniciushammett/go-log-anomaly-detector/internal/notify"
	"github.com/viniciushammett/go-log-anomaly-detector/internal/rules"
	"github.com/viniciushammett/go-log-anomaly-detector/internal/store"
	"github.com/viniciushammett/go-log-anomaly-detector/internal/util"
)

type Detector struct {
	log   *logger.Logger
	db    *store.Store
	rs    *rules.Set
	slack *notify.Slack

	win map[string]*util.Sliding // janela por regra
}

func New(log *logger.Logger, db *store.Store, rs *rules.Set, slack *notify.Slack) *Detector {
	return &Detector{log: log, db: db, rs: rs, slack: slack, win: map[string]*util.Sliding{}}
}

type LogLine struct {
	TS     time.Time
	Source string
	Msg    string
	Meta   map[string]string
}

func (d *Detector) Submit(l LogLine) {
	if l.TS.IsZero() { l.TS = time.Now() }
	metrics.LogsIngested.WithLabelValues(l.Source).Inc()
	for _, r := range d.rs.Items {
		if r.RE.MatchString(l.Msg) {
			s := d.win[r.Name]
			if s == nil { s = &util.Sliding{}; d.win[r.Name] = s }
			s.Add(util.Point{TS:l.TS, Data:l.Msg}, r.Window)
			metrics.WindowGauge.WithLabelValues(r.Name).Set(float64(s.Count()))
			// threshold por janela
			if r.MinCount > 0 && s.Count() >= r.MinCount {
				d.raise("threshold", r, s.Count(), l)
				// reset simples para evitar chuva
				d.win[r.Name] = &util.Sliding{}
			}
		}
	}
}

func (d *Detector) raise(kind string, r rules.Rule, count int, l LogLine) {
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