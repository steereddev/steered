package client

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Client holds the kubernetes clientset and cluster info
type Client struct {
	Kubernetes  kubernetes.Interface
	ClusterName string
	Context     string
}

// New creates a new Client following kubectl kubeconfig precedence:
// 1. explicit --kubeconfig flag
// 2. KUBECONFIG env var
// 3. ~/.kube/config default
func New(kubeconfigPath, context string) (*Client, error) {
	kubeconfig := resolveKubeconfig(kubeconfigPath)

	// load rules following standard kubectl precedence
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = kubeconfig

	configOverrides := &clientcmd.ConfigOverrides{}
	if context != "" {
		configOverrides.CurrentContext = context
	}

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		configOverrides,
	)

	// get current context name for display
	rawConfig, err := kubeConfig.RawConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	currentContext := rawConfig.CurrentContext
	if context != "" {
		currentContext = context
	}

	// get cluster name from context
	clusterName := ""
	if ctx, ok := rawConfig.Contexts[currentContext]; ok {
		clusterName = ctx.Cluster
	}

	// build rest config
	restConfig, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to build rest config: %w", err)
	}

	// build clientset
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &Client{
		Kubernetes:  clientset,
		ClusterName: clusterName,
		Context:     currentContext,
	}, nil
}

// resolveKubeconfig returns the kubeconfig path to use
func resolveKubeconfig(explicit string) string {
	// 1. explicit flag
	if explicit != "" {
		return explicit
	}

	// 2. KUBECONFIG env var
	if env := os.Getenv("KUBECONFIG"); env != "" {
		return env
	}

	// 3. default ~/.kube/config
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".kube", "config")
}
