package k8s

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	typed "k8s.io/client-go/kubernetes/typed/apps/v1"
)

type Deployer struct {
	cs typed.DeploymentInterface
}

func NewDeployer(cs typed.DeploymentInterface) *Deployer { return &Deployer{cs: cs} }

func (d *Deployer) Get(ctx context.Context, name string) (*appsv1.Deployment, error) {
	return d.cs.Get(ctx, name, meta.GetOptions{})
}

func (d *Deployer) SetImage(ctx context.Context, name, container, image string) error {
	dep, err := d.Get(ctx, name); if err != nil { return err }
	found := false
	for i := range dep.Spec.Template.Spec.Containers {
		if dep.Spec.Template.Spec.Containers[i].Name == container {
			dep.Spec.Template.Spec.Containers[i].Image = image
			found = true
		}
	}
	if !found {
		// assume primeiro container
		if len(dep.Spec.Template.Spec.Containers) == 0 {
			dep.Spec.Template.Spec.Containers = []corev1.Container{{Name: container, Image: image}}
		} else {
			dep.Spec.Template.Spec.Containers[0].Image = image
		}
	}
	_, err = d.cs.Update(ctx, dep, meta.UpdateOptions{})
	return err
}

func (d *Deployer) Scale(ctx context.Context, name string, replicas int32) error {
	dep, err := d.Get(ctx, name); if err != nil { return err }
	dep.Spec.Replicas = &replicas
	_, err = d.cs.Update(ctx, dep, meta.UpdateOptions{})
	return err
}

func (d *Deployer) WaitRollout(ctx context.Context, name string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		dep, err := d.Get(ctx, name); if err != nil { return err }
		if dep.Status.UpdatedReplicas == *dep.Spec.Replicas &&
			dep.Status.ReadyReplicas == *dep.Spec.Replicas &&
			dep.Status.ObservedGeneration >= dep.Generation {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("rollout timeout for %s", name)
}