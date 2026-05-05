package collector

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/steereddev/steered/internal/client"
	"github.com/steereddev/steered/internal/model"
)

// SecretCollector collects secret information from the cluster
type SecretCollector struct {
	client *client.Client
}

// NewSecretCollector creates a new SecretCollector
func NewSecretCollector(c *client.Client) *SecretCollector {
	return &SecretCollector{client: c}
}

// Collect gathers all secret data and fills snapshot.Secrets
func (d *SecretCollector) Collect(ctx context.Context, snapshot *model.ClusterSnapshot) error {
	list, err := d.client.Kubernetes.CoreV1().Secrets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to collect secrets: %w", err)
	}

	for _, s := range list.Items {
		// skip system secrets
		if isSystemNamespace(s.Namespace) {
			continue
		}
		snapshot.Secrets = append(snapshot.Secrets, model.Secret{
			Name:      s.Name,
			Namespace: s.Namespace,
		})
	}

	return nil
}
