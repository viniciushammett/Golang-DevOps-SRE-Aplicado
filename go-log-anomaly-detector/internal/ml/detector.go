package ml

import (
	"encoding/json"
	"os"
	"time"
	"sync"
)

type stats struct {
	Mean float64 `json:"mean"`
	Std  float64 `json:"std"`
}

type Model struct {
	// chave -> stats
	Keys map[string]stats `json:"keys"`
}

type Detector struct {
	mu     sync.Mutex
	model  Model
	k      float64
	bucket time.Duration
	// contadores por chave e janela atual
	counts map[string]int
	winKey string // timestamp da janela atual
}

func NewDetector(modelPath string, zK float64, bucket string) (*Detector, error) {
	if bucket == "" { bucket = "1m" }
	dur, _ := time.ParseDuration(bucket)
	if dur == 0 { dur = time.Minute }
	var m Model
	if b, err := os.ReadFile(modelPath); err == nil {
		_ = json.Unmarshal(b, &m)
	}
	return &Detector{
		model: m, k: zK, bucket: dur,
		counts: map[string]int{},
	}, nil
}

// chave padrão: por source; pode customizar (ex.: regex clusterizada)
func keyFor(source string) string { if source=="" { return "unknown" }; return "src="+source }

func (d *Detector) Observe(line struct{
	TS time.Time; Source, Msg string; Meta map[string]string,
}) (anom bool, score float64, count int, key string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := line.TS
	b := now.Truncate(d.bucket)
	if b.String() != d.winKey {
		// troca de janela → limpa contadores
		d.counts = map[string]int{}
		d.winKey = b.String()
	}
	key = keyFor(line.Source)
	d.counts[key]++
	count = d.counts[key]

	// lookup stats
	st := d.model.Keys[key]
	if st.Std <= 0 { return false, 0, count, key }
	thr := st.Mean + d.k*st.Std
	if float64(count) > thr {
		score = (float64(count) - st.Mean) / st.Std
		return true, score, count, key
	}
	return false, 0, count, key
}