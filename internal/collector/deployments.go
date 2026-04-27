package collector

import (
	"context"
	"fmt"

	"github.com/steereddev/steered/internal/client"
	"github.com/steereddev/steered/internal/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeploymentCollector collects deployment information from the cluster
type DeploymentCollector struct {
	client *client.Client
}

// NewDeploymentCollector creates a new DeploymentCollector
func NewDeploymentCollector(c *client.Client) *DeploymentCollector {
	return &DeploymentCollector{client: c}
}

// Collect gathers all deployment data and fills snapshot.Deployments
func (d *DeploymentCollector) Collect(ctx context.Context, snapshot *model.ClusterSnapshot) error {
	deployments, err := d.client.Kubernetes.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to collect deployments: %w", err)
	}

	for _, deploy := range deployments.Items {
		// check if resource limits are set on any container
		hasLimits := true
		for _, c := range deploy.Spec.Template.Spec.Containers {
			if c.Resources.Limits == nil || len(c.Resources.Limits) == 0 {
				hasLimits = false
				break
			}
		}

		// if no limits flag for cost signals
		if !hasLimits {
			snapshot.CostSignals.PodsWithNoLimits = append(
				snapshot.CostSignals.PodsWithNoLimits,
				fmt.Sprintf("%s/%s", deploy.Namespace, deploy.Name),
			)
		}

		snapshot.Deployments = append(snapshot.Deployments, model.Deployment{
			Name:      deploy.Name,
			Namespace: deploy.Namespace,
			Ready:     fmt.Sprintf("%d/%d", deploy.Status.ReadyReplicas, deploy.Status.Replicas),
			UpToDate:  deploy.Status.UpdatedReplicas,
			Available: deploy.Status.AvailableReplicas,
			Age:       age(deploy.CreationTimestamp.Time),
		})
	}

	return nil
}
