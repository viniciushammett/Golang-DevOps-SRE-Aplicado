package orchestrator

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/viniciushammett/go-deploy-orchestrator/internal/config"
	"github.com/viniciushammett/go-deploy-orchestrator/internal/k8s"
	"github.com/viniciushammett/go-deploy-orchestrator/internal/logger"
	"github.com/viniciushammett/go-deploy-orchestrator/internal/metrics"
	"github.com/viniciushammett/go-deploy-orchestrator/internal/prometheus"
	"github.com/viniciushammett/go-deploy-orchestrator/internal/store"
	"github.com/viniciushammett/go-deploy-orchestrator/internal/strategies"
	"k8s.io/client-go/kubernetes"
)

type Orchestrator struct {
	log  *logger.Logger
	cfg  *config.Config
	db   *store.Store
	prom *prometheus.Evaluator
	kcs  *kubernetes.Clientset
}

func New(log *logger.Logger, cfg *config.Config, db *store.Store, prom *prometheus.Evaluator) (*Orchestrator, error) {
	cs, err := k8s.NewClient(cfg.Kube.Kubeconfig, cfg.Kube.Context)
	if err != nil { return nil, err }
	return &Orchestrator{log: log, cfg: cfg, db: db, prom: prom, kcs: cs}, nil
}

type DeployInput struct {
	App       string
	Namespace string
	Image     string
	Strategy  string
	Params    map[string]string
	RequireApproval bool
}

func (o *Orchestrator) StartDeploy(ctx context.Context, in DeployInput) (*store.DeployRecord, error) {
	id := randID()
	rec := store.DeployRecord{
		ID: id, App: in.App, Namespace: in.Namespace, ImageNew: in.Image, Strategy: in.Strategy,
		Status: "started", StartedAt: time.Now(), Params: in.Params,
	}
	if err := o.db.Put(rec); err != nil { return nil, err }

	metrics.DeploysStarted.WithLabelValues(in.App, in.Strategy).Inc()

	go o.run(ctx, rec, in.RequireApproval)
	return &rec, nil
}

func (o *Orchestrator) run(ctx context.Context, rec store.DeployRecord, requireApproval bool) {
	ns := rec.Namespace
	if ns == "" { ns = "default" }

	deployIf := o.kcs.AppsV1().Deployments(ns)
	dep := k8s.NewDeployer(deployIf)

	// snapshot do image atual
	orig, err := dep.Get(ctx, rec.App)
	if err == nil && len(orig.Spec.Template.Spec.Containers) > 0 {
		rec.ImageOld = orig.Spec.Template.Spec.Containers[0].Image
		_ = o.db.Put(rec)
	}

	if requireApproval {
		rec.Status = "waiting_approval"
		_ = o.db.Put(rec)
		// aguarda sinal externo via API /approve (para simplificar, só muda status)
		for {
			time.Sleep(1 * time.Second)
			cur, _ := o.db.Get(rec.ID)
			if cur != nil && cur.Status == "running" { break }
			select { case <-ctx.Done(): return default: }
		}
	}

	// executa a estratégia
	err = o.applyStrategy(ctx, dep, rec)
	if err != nil {
		metrics.DeploysFailed.WithLabelValues(rec.App, rec.Strategy, "strategy_error").Inc()
		o.rollback(ctx, dep, rec, err)
		return
	}

	now := time.Now()
	rec.Status = "succeeded"
	rec.FinishedAt = &now
	_ = o.db.Put(rec)
	metrics.DeploysSucceeded.WithLabelValues(rec.App, rec.Strategy).Inc()
}

func (o *Orchestrator) applyStrategy(ctx context.Context, dep *k8s.Deployer, rec store.DeployRecord) error {
	switch rec.Strategy {
	case "canary":
		step, _ := atoi(rec.Params["canaryStep"])
		if step == 0 { step = o.cfg.Defaults.CanaryStepPercent }
		pause, _ := atoi(rec.Params["canaryPause"])
		if pause == 0 { pause = o.cfg.Defaults.CanaryPauseSec }
		maxError, _ := atof(rec.Params["maxError"])
		if maxError == 0 { maxError = o.cfg.Prometheus.Thresholds.MaxError }
		maxP95, _ := atof(rec.Params["maxP95"])
		if maxP95 == 0 { maxP95 = o.cfg.Prometheus.Thresholds.MaxP95 }
		return strategies.RunCanary(ctx, dep, o.prom, rec.App, rec.ImageNew, strategies.CanaryParams{
			StepPercent: step, PauseSec: pause, MaxError: maxError, MaxP95: maxP95,
		})
	case "bluegreen":
		wait, _ := atoi(rec.Params["probeWait"])
		maxError, _ := atof(rec.Params["maxError"])
		if maxError == 0 { maxError = o.cfg.Prometheus.Thresholds.MaxError }
		maxP95, _ := atof(rec.Params["maxP95"])
		if maxP95 == 0 { maxP95 = o.cfg.Prometheus.Thresholds.MaxP95 }
		return strategies.RunBlueGreen(ctx, dep, o.prom, rec.App, rec.ImageNew, strategies.BlueGreenParams{
			ProbeWaitSec: wait, MaxError: maxError, MaxP95: maxP95,
		})
	default:
		return fmt.Errorf("unknown strategy %q", rec.Strategy)
	}
}

func (o *Orchestrator) rollback(ctx context.Context, dep *k8s.Deployer, rec store.DeployRecord, err error) {
	o.log.Error().Err(err).Str("app", rec.App).Msg("deploy failed, rolling back")
	if rec.ImageOld != "" {
		_ = dep.SetImage(ctx, rec.App, rec.App, rec.ImageOld)
		_ = dep.WaitRollout(ctx, rec.App, 5*time.Minute)
	}
	now := time.Now()
	rec.Status = "rolled_back"
	rec.Reason = err.Error()
	rec.FinishedAt = &now
	_ = o.db.Put(rec)
}

func randID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
func atoi(s string) (int, error) { return strconv.Atoi(s) }
func atof(s string) (float64, error) { return strconv.ParseFloat(s, 64) }