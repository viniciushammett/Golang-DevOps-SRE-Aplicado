package k8s

import (
	"flag"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func NewClient(kubeconfig, context string) (*kubernetes.Clientset, error) {
	if cfg, err := rest.InClusterConfig(); err == nil {
		return kubernetes.NewForConfig(cfg)
	}
	if kubeconfig == "" {
		if home, err := os.UserHomeDir(); err == nil {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	}
	_ = flag.CommandLine.Parse([]string{})
	loading := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig}
	over := &clientcmd.ConfigOverrides{}
	if context != "" { over.CurrentContext = context }
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loading, over).ClientConfig()
	if err != nil { return nil, err }
	return kubernetes.NewForConfig(cfg)
}