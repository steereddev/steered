package collector

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/steereddev/steered/internal/client"
	"github.com/steereddev/steered/internal/model"
)

// ConfigMapCollector collects configmap information from the cluster
type ConfigMapCollector struct {
	client *client.Client
}

// NewConfigMapCollector creates a new ConfigMapCollector
func NewConfigMapCollector(c *client.Client) *ConfigMapCollector {
	return &ConfigMapCollector{client: c}
}

// Collect gathers all configmap data and fills snapshot.ConfigMaps
func (d *ConfigMapCollector) Collect(ctx context.Context, snapshot *model.ClusterSnapshot) error {
	list, err := d.client.Kubernetes.CoreV1().ConfigMaps("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to collect configmaps: %w", err)
	}

	for _, cm := range list.Items {
		// skip system configmaps
		if isSystemNamespace(cm.Namespace) {
			continue
		}
		snapshot.ConfigMaps = append(snapshot.ConfigMaps, model.ConfigMap{
			Name:      cm.Name,
			Namespace: cm.Namespace,
		})
	}

	return nil
}
