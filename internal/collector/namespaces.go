package collector

import (
	"context"
	"fmt"

	"github.com/steereddev/steered/internal/client"
	"github.com/steereddev/steered/internal/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NamespaceCollector collects namespace information from the cluster
type NamespaceCollector struct {
	client *client.Client
}

// NewNamespaceCollector creates a new NamespaceCollector
func NewNamespaceCollector(c *client.Client) *NamespaceCollector {
	return &NamespaceCollector{client: c}
}

// Collect gathers all namespace data and fills snapshot.Namespaces
func (n *NamespaceCollector) Collect(ctx context.Context, snapshot *model.ClusterSnapshot) error {
	namespaces, err := n.client.Kubernetes.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to collect namespaces: %w", err)
	}

	for _, ns := range namespaces.Items {
		snapshot.Namespaces = append(snapshot.Namespaces, model.Namespace{
			Name:   ns.Name,
			Status: string(ns.Status.Phase),
			Age:    age(ns.CreationTimestamp.Time),
		})
	}

	return nil
}
