package ingest

import (
	"time"

	"github.com/viniciushammett/go-log-anomaly-detector/internal/detector"
)

type Ingest struct {
	det *detector.Detector
}

func New(det *detector.Detector) *Ingest { return &Ingest{det: det} }

// HTTP/JSON chama isto:
func (i *Ingest) Submit(source, msg string, meta map[string]string, ts time.Time) {
	i.det.Submit(detector.LogLine{TS: ts, Source: source, Msg: msg, Meta: meta})
}

// Stubs para Kafka/File watchers — implementar conforme sua infra (plugins):
func (i *Ingest) FromKafka(topic string) {}
func (i *Ingest) FromFile(path string)  {}