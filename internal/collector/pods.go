package collector

import (
	"context"
	"fmt"
	"strings"

	"github.com/steereddev/steered/internal/client"
	"github.com/steereddev/steered/internal/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodCollector collects pod information from the cluster
type PodCollector struct {
	client *client.Client
}

// NewPodCollector creates a new PodCollector
func NewPodCollector(c *client.Client) *PodCollector {
	return &PodCollector{client: c}
}

// Collect gathers all pod data and fills snapshot.Pods
func (p *PodCollector) Collect(ctx context.Context, snapshot *model.ClusterSnapshot) error {
	pods, err := p.client.Kubernetes.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to collect pods: %w", err)
	}

	for _, pod := range pods.Items {
		var restarts int32
		var ready int32

		// collect raw container states
		var containerStates []model.ContainerState
		for _, cs := range pod.Status.ContainerStatuses {
			restarts += cs.RestartCount
			if cs.Ready {
				ready++
			}

			state := model.ContainerState{
				Name:     cs.Name,
				Ready:    cs.Ready,
				Restarts: cs.RestartCount,
			}

			// waiting state — captures ImagePullBackOff, CrashLoopBackOff etc
			if cs.State.Waiting != nil {
				state.WaitingReason = cs.State.Waiting.Reason
				state.WaitingMessage = cs.State.Waiting.Message
			}

			// terminated state — captures OOMKilled, Error etc
			if cs.State.Terminated != nil {
				state.TerminatedReason = cs.State.Terminated.Reason
				state.ExitCode = cs.State.Terminated.ExitCode
			}

			// last termination state
			if cs.LastTerminationState.Terminated != nil {
				if state.TerminatedReason == "" {
					state.TerminatedReason = cs.LastTerminationState.Terminated.Reason
					state.ExitCode = cs.LastTerminationState.Terminated.ExitCode
				}
			}

			containerStates = append(containerStates, state)
		}

		totalContainers := int32(len(pod.Spec.Containers))

		// resource requests and limits
		cpuRequest := ""
		memRequest := ""
		cpuLimit := ""
		memLimit := ""

		if len(pod.Spec.Containers) > 0 {
			c := pod.Spec.Containers[0]
			if req := c.Resources.Requests; req != nil {
				if cpu := req.Cpu(); cpu != nil {
					cpuRequest = cpu.String()
				}
				if mem := req.Memory(); mem != nil {
					memRequest = mem.String()
				}
			}
			if lim := c.Resources.Limits; lim != nil {
				if cpu := lim.Cpu(); cpu != nil {
					cpuLimit = cpu.String()
				}
				if mem := lim.Memory(); mem != nil {
					memLimit = mem.String()
				}
			}
		}

		// owner reference
		ownerKind := ""
		ownerName := ""
		if len(pod.OwnerReferences) > 0 {
			owner := pod.OwnerReferences[0]
			if owner.Kind == "ReplicaSet" {
				parts := strings.Split(owner.Name, "-")
				if len(parts) > 1 {
					ownerKind = "deployment"
					ownerName = strings.Join(parts[:len(parts)-1], "-")
				}
			} else {
				ownerKind = strings.ToLower(owner.Kind)
				ownerName = owner.Name
			}
		}

		snapshot.Pods = append(snapshot.Pods, model.Pod{
			Name:            pod.Name,
			Namespace:       pod.Namespace,
			Status:          string(pod.Status.Phase),
			Ready:           fmt.Sprintf("%d/%d", ready, totalContainers),
			Restarts:        restarts,
			Age:             age(pod.CreationTimestamp.Time),
			Node:            pod.Spec.NodeName,
			OwnerKind:       ownerKind,
			OwnerName:       ownerName,
			CPURequest:      cpuRequest,
			MemoryRequest:   memRequest,
			CPULimit:        cpuLimit,
			MemoryLimit:     memLimit,
			ContainerStates: containerStates,
		})
	}

	return nil
}
