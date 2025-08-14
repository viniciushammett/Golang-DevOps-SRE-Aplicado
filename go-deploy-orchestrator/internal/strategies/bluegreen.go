package strategies

import (
	"context"
	"fmt"
	"time"

	"github.com/viniciushammett/go-deploy-orchestrator/internal/k8s"
	"github.com/viniciushammett/go-deploy-orchestrator/internal/prometheus"
)

type BlueGreenParams struct {
	ProbeWaitSec int
	MaxError     float64
	MaxP95       float64
}

func RunBlueGreen(ctx context.Context, dep *k8s.Deployer, prom *prometheus.Evaluator, app, image string, p BlueGreenParams) error {
	if p.ProbeWaitSec == 0 { p.ProbeWaitSec = 30 }
	// update image & rollout all replicas (blue->green swap simplificada: troca de template)
	if err := dep.SetImage(ctx, app, app, image); err != nil { return err }
	if err := dep.WaitRollout(ctx, app, 5*time.Minute); err != nil { return err }

	time.Sleep(time.Duration(p.ProbeWaitSec) * time.Second)

	// checa SLOs (placeholder seguro)
	er, _ := prom.QueryRange(ctx, `vector(0)`, "5m")
	p95, _ := prom.QueryRange(ctx, `vector(0)`, "5m")
	if (p.MaxError > 0 && er > p.MaxError) || (p.MaxP95 > 0 && p95 > p.MaxP95) {
		return fmt.Errorf("SLO breach after blue-green (error=%.4f p95=%.3fs)", er, p95)
	}
	return nil
}
// RunBlueGreen executes a blue-green deployment strategy.
// It updates the image of the application, waits for the rollout to complete.