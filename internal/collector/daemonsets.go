package collector

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/steereddev/steered/internal/client"
	"github.com/steereddev/steered/internal/model"
)

// DaemonSetCollector collects daemonset information from the cluster
type DaemonSetCollector struct {
	client *client.Client
}

// NewDaemonSetCollector creates a new DaemonSetCollector
func NewDaemonSetCollector(c *client.Client) *DaemonSetCollector {
	return &DaemonSetCollector{client: c}
}

// Collect gathers all daemonset data and fills snapshot.DaemonSets
func (d *DaemonSetCollector) Collect(ctx context.Context, snapshot *model.ClusterSnapshot) error {
	list, err := d.client.Kubernetes.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to collect daemonsets: %w", err)
	}

	for _, ds := range list.Items {
		snapshot.DaemonSets = append(snapshot.DaemonSets, model.DaemonSet{
			Name:      ds.Name,
			Namespace: ds.Namespace,
			Desired:   ds.Status.DesiredNumberScheduled,
			Current:   ds.Status.CurrentNumberScheduled,
			Ready:     ds.Status.NumberReady,
			UpToDate:  ds.Status.UpdatedNumberScheduled,
			Available: ds.Status.NumberAvailable,
			Age:       age(ds.CreationTimestamp.Time),
		})
	}

	return nil
}
