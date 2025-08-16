package ingest

import (
	"time"

	"github.com/viniciushammett/go-log-anomaly-detector/internal/detector"
	"github.com/viniciushammett/go-log-anomaly-detector/internal/store"
)

type Ingest struct {
	det *detector.Detector
	st  *store.Store
}

func New(det *detector.Detector, st *store.Store) *Ingest { // <-- passa store
	return &Ingest{det: det, st: st}
}

func (i *Ingest) Submit(source, msg string, meta map[string]string, ts time.Time) {
	if ts.IsZero() { ts = time.Now() }
	// persistir log bruto
	if i.st != nil {
		_ = i.st.PutLog(store.LogRecord{TS: ts, Source: source, Msg: msg, Meta: meta})
	}
	// enviar para detecção
	i.det.Submit(detector.LogLine{TS: ts, Source: source, Msg: msg, Meta: meta})
}

// Stubs
func (i *Ingest) FromKafka(topic string) {}
func (i *Ingest) FromFile(path string)  {}
