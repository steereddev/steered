package collector

import (
	"context"
	"fmt"
	"strings"

	"github.com/steereddev/steered/internal/client"
	"github.com/steereddev/steered/internal/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PVCCollector collects persistent volume claim information from the cluster
type PVCCollector struct {
	client *client.Client
}

// NewPVCCollector creates a new PVCCollector
func NewPVCCollector(c *client.Client) *PVCCollector {
	return &PVCCollector{client: c}
}

// Collect gathers all pvc data and fills snapshot.PVCs
func (p *PVCCollector) Collect(ctx context.Context, snapshot *model.ClusterSnapshot) error {
	pvcs, err := p.client.Kubernetes.CoreV1().PersistentVolumeClaims("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to collect pvcs: %w", err)
	}

	// get all pods to check which pvcs are mounted
	pods, err := p.client.Kubernetes.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to collect pods for pvc check: %w", err)
	}

	// build set of mounted pvcs
	mountedPVCs := make(map[string]bool)
	for _, pod := range pods.Items {
		for _, vol := range pod.Spec.Volumes {
			if vol.PersistentVolumeClaim != nil {
				key := fmt.Sprintf("%s/%s", pod.Namespace, vol.PersistentVolumeClaim.ClaimName)
				mountedPVCs[key] = true
			}
		}
	}

	for _, pvc := range pvcs.Items {
		// collect access modes
		var accessModes []string
		for _, mode := range pvc.Spec.AccessModes {
			accessModes = append(accessModes, string(mode))
		}

		// get storage class
		storageClass := ""
		if pvc.Spec.StorageClassName != nil {
			storageClass = *pvc.Spec.StorageClassName
		}

		// get capacity
		capacity := ""
		if pvc.Status.Capacity != nil {
			if storage := pvc.Status.Capacity.Storage(); storage != nil {
				capacity = storage.String()
			}
		}

		// check if unattached — flag as cost signal
		key := fmt.Sprintf("%s/%s", pvc.Namespace, pvc.Name)
		if !mountedPVCs[key] && string(pvc.Status.Phase) == "Bound" {
			snapshot.CostSignals.UnattachedPVCs = append(
				snapshot.CostSignals.UnattachedPVCs,
				key,
			)
		}

		snapshot.PVCs = append(snapshot.PVCs, model.PVC{
			Name:         pvc.Name,
			Namespace:    pvc.Namespace,
			Status:       string(pvc.Status.Phase),
			Volume:       pvc.Spec.VolumeName,
			Capacity:     capacity,
			AccessModes:  strings.Join(accessModes, ", "),
			StorageClass: storageClass,
			Age:          age(pvc.CreationTimestamp.Time),
		})
	}

	return nil
}
