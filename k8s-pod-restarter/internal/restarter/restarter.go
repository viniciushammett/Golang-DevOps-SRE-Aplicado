package restarter

import (
	"context"
	"time"

	"github.com/youruser/k8s-pod-restarter/internal/logger"
	"github.com/youruser/k8s-pod-restarter/internal/metrics"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type Options struct {
	Namespace   string
	Selector    string
	DryRun      bool
	Force       bool
	MaxAge      time.Duration
	GracePeriod time.Duration
}

type Restarter struct {
	log *logger.Logger
	cs  *kubernetes.Clientset
}

func New(log *logger.Logger, cs *kubernetes.Clientset) *Restarter {
	return &Restarter{log: log, cs: cs}
}

// RestartPods deletes pods that match the selector (optionally namespace-scoped).
// Controllers (Deployment/ReplicaSet/StatefulSet/DaemonSet) will recreate them.
func (r *Restarter) RestartPods(ctx context.Context, o Options) error {
	start := time.Now()
	ns := o.Namespace
	if ns == "" {
		ns = metav1.NamespaceAll
	}

	ls, err := labels.Parse(o.Selector)
	if err != nil {
		return err
	}

	pods, err := r.cs.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
		LabelSelector: ls.String(),
		FieldSelector: fields.Everything().String(),
	})
	if err != nil {
		metrics.ErrorsTotal.WithLabelValues(o.Namespace, "list").Inc()
		return err
	}

	var deleted int
	reason := "manual"
	for _, p := range pods.Items {
		if o.MaxAge > 0 && p.CreationTimestamp.Add(o.MaxAge).After(time.Now()) {
			continue // too new
		}
		if !o.Force && isDaemonOrStatic(&p) {
			continue
		}
		r.log.Info().Str("ns", p.Namespace).Str("pod", p.Name).Msg("deleting pod")
		if !o.DryRun {
			grace := int64(o.GracePeriod.Seconds())
			err := r.cs.CoreV1().Pods(p.Namespace).Delete(ctx, p.Name, metav1.DeleteOptions{
				GracePeriodSeconds: &grace,
			})
			if err != nil {
				r.log.Error().Err(err).Str("ns", p.Namespace).Str("pod", p.Name).Msg("delete failed")
				metrics.ErrorsTotal.WithLabelValues(p.Namespace, "delete").Inc()
				continue
			}
		}
		deleted++
		metrics.RestartsTotal.WithLabelValues(p.Namespace, reason).Inc()
		metrics.LastRestart.WithLabelValues(p.Namespace).Set(float64(time.Now().Unix()))
	}

	metrics.OpDuration.WithLabelValues(o.Namespace).Observe(time.Since(start).Seconds())
	r.log.Info().Int("pods_deleted", deleted).Str("namespace", o.Namespace).Msg("restart operation done")
	return nil
}

func isDaemonOrStatic(p *corev1.Pod) bool {
	// Static pod: has annotation kubelet.kubernetes.io/config.source
	if _, ok := p.Annotations["kubelet.kubernetes.io/config.source"]; ok {
		return true
	}
	// DaemonSet-owned pod
	for _, ref := range p.OwnerReferences {
		if ref.Kind == "DaemonSet" {
			return true
		}
	}
	return false
}