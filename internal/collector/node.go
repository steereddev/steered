package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/steereddev/steered/internal/client"
	"github.com/steereddev/steered/internal/model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NodeCollector collects node information from the cluster
type NodeCollector struct {
	client *client.Client
}

// NewNodeCollector creates a new NodeCollector
func NewNodeCollector(c *client.Client) *NodeCollector {
	return &NodeCollector{client: c}
}

// Collect gathers all node data and fills snapshot.Nodes
func (n *NodeCollector) Collect(ctx context.Context, snapshot *model.ClusterSnapshot) error {
	nodes, err := n.client.Kubernetes.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to collect nodes: %w", err)
	}

	for _, node := range nodes.Items {
		snapshot.Nodes = append(snapshot.Nodes, model.Node{
			Name:           node.Name,
			Status:         nodeStatus(node),
			Roles:          nodeRoles(node),
			Age:            age(node.CreationTimestamp.Time),
			Version:        node.Status.NodeInfo.KubeletVersion,
			CPUCapacity:    node.Status.Capacity.Cpu().String(),
			MemoryCapacity: memoryToGi(node),
		})
	}

	return nil
}

// nodeStatus returns Ready or NotReady based on node conditions
func nodeStatus(node corev1.Node) string {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				return "Ready"
			}
			return "NotReady"
		}
	}
	return "Unknown"
}

// nodeRoles returns the roles of a node from its labels
func nodeRoles(node corev1.Node) string {
	roles := ""
	if _, ok := node.Labels["node-role.kubernetes.io/control-plane"]; ok {
		roles = "control-plane"
	} else if _, ok := node.Labels["node-role.kubernetes.io/master"]; ok {
		roles = "master"
	} else {
		roles = "worker"
	}
	return roles
}

// age returns a human readable age string from a time
func age(t time.Time) string {
	d := time.Since(t)
	if d.Hours() > 48 {
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
	if d.Hours() > 1 {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dm", int(d.Minutes()))
}

// memoryToGi converts memory quantity to Gi string
func memoryToGi(node corev1.Node) string {
	mem := node.Status.Capacity.Memory()
	gi := float64(mem.Value()) / (1024 * 1024 * 1024)
	return fmt.Sprintf("%.1fGi", gi)
}
