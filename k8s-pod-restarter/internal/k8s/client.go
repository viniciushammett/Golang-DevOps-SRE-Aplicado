package k8s

import (
	"flag"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func NewClient(kubeconfigPath, contextName string) (*kubernetes.Clientset, error) {
	// In-cluster first
	if cfg, err := rest.InClusterConfig(); err == nil {
		return kubernetes.NewForConfig(cfg)
	}

	// Out-of-cluster (dev)
	if kubeconfigPath == "" {
		if home, err := os.UserHomeDir(); err == nil {
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		}
	}
	// client-go uses flags internally; avoid side effects
	_ = flag.CommandLine.Parse([]string{})

	loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath}
	overrides := &clientcmd.ConfigOverrides{}
	if contextName != "" {
		overrides.CurrentContext = contextName
	}
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides).ClientConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(cfg)
}