package strategies

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/viniciushammett/go-deploy-orchestrator/internal/k8s"
	"github.com/viniciushammett/go-deploy-orchestrator/internal/metrics"
	"github.com/viniciushammett/go-deploy-orchestrator/internal/prometheus"
)

type CanaryParams struct {
	StepPercent int
	PauseSec    int
	MaxError    float64
	MaxP95      float64
}

func RunCanary(ctx context.Context, dep *k8s.Deployer, prom *prometheus.Evaluator, app string, image string, params CanaryParams) error {
	if params.StepPercent <= 0 { params.StepPercent = 20 }
	if params.PauseSec <= 0 { params.PauseSec = 60 }

	// set image
	if err := dep.SetImage(ctx, app, app, image); err != nil { return err }

	// discover replicas
	d, err := dep.Get(ctx, app); if err != nil { return err }
	replicas := int32(1)
	if d.Spec.Replicas != nil { replicas = *d.Spec.Replicas }

	// progressive traffic: emulate by scaling up in steps (simple approach)
	step := int32(max(1, (int(replicas) * params.StepPercent / 100)))
	for cur := step; cur <= replicas; cur += step {
		if cur > replicas { cur = replicas }
		if err := dep.Scale(ctx, app, cur); err != nil { return err }
		if err := dep.WaitRollout(ctx, app, 5*time.Minute); err != nil { return err }
		metrics.StepDuration.WithLabelValues(app, "canary", "scale_"+itoa(int(cur))).Observe(float64(params.PauseSec))

		// check SLOs (if configured)
		ok, er, p95 := passSLOs(ctx, prom, params)
		if !ok {
			return fmt.Errorf("SLO breach during canary (error=%.4f p95=%.3fs)", er, p95)
		}
		time.Sleep(time.Duration(params.PauseSec) * time.Second)
	}
	return nil
}

func passSLOs(ctx context.Context, prom *prometheus.Evaluator, p CanaryParams) (bool, float64, float64) {
	er, _ := prom.QueryRange(ctx, `vector(0) + on() group_left() 0`, "5m") // placeholder safe
	p95, _ := prom.QueryRange(ctx, `vector(0) + on() group_left() 0`, "5m")
	// se tiver thresholds > 0, valida
	if p.MaxError > 0 && er > p.MaxError { return false, er, p95 }
	if p.MaxP95 > 0 && p95 > p.MaxP95 { return false, er, p95 }
	return true, er, p95
}

func max(a,b int) int { if a>b {return a}; return b }
func itoa(v int) string { return strconv.Itoa(v) }