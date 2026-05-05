package context

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/steereddev/steered/internal/client"
	"github.com/steereddev/steered/internal/llm"
	"github.com/steereddev/steered/internal/model"
)

// Builder fetches rich context per resource from the cluster
type Builder struct {
	client *client.Client
}

// New creates a new context Builder
func New(c *client.Client) *Builder {
	return &Builder{client: c}
}

// Build fetches rich context for a resource
func (b *Builder) Build(ctx context.Context, snapshot *model.ClusterSnapshot, resourceName, namespace, kind string) (*llm.IssueContext, error) {
	ic := &llm.IssueContext{
		ResourceName:      resourceName,
		ResourceNamespace: namespace,
		ResourceKind:      kind,
		ClusterName:       snapshot.ClusterName,
		Identifiers:       make(map[string]string),
	}

	ic.Identifiers["RESOURCE_KIND"] = kind
	ic.Identifiers["RESOURCE_NAME"] = resourceName
	ic.Identifiers["NAMESPACE"] = namespace

	// load policy context from skills file
	loader := llm.NewPromptLoader()
	ic.PolicyContext = loader.LoadIssue(kind)

	// always append security CVE guidance
	cveContext := loader.LoadSecurity()
	if cveContext != "" {
		ic.PolicyContext += "\n\n" + cveContext
	}

	switch strings.ToLower(kind) {
	case "deployment":
		b.enrichDeploymentContext(ctx, ic, snapshot)
	case "pod":
		b.enrichPodContext(ctx, ic)
	case "node":
		b.enrichNodeContext(ctx, ic)
	case "namespace":
		b.enrichNamespaceContext(ctx, ic)
	case "ingress":
		b.enrichIngressContext(ctx, ic)
	case "pvc":
		b.enrichPVCContext(ctx, ic)
	case "service":
		b.enrichServiceContext(ctx, ic)
	default:
		b.enrichGenericContext(ctx, ic)
	}

	return ic, nil
}

// enrichDeploymentContext fetches full context for a deployment
func (b *Builder) enrichDeploymentContext(ctx context.Context, ic *llm.IssueContext, snapshot *model.ClusterSnapshot) {
	deploy, err := b.client.Kubernetes.AppsV1().
		Deployments(ic.ResourceNamespace).
		Get(ctx, ic.ResourceName, metav1.GetOptions{})
	if err != nil {
		return
	}

	// replica count
	if deploy.Spec.Replicas != nil {
		ic.Identifiers["REPLICA_COUNT"] = fmt.Sprintf("%d", *deploy.Spec.Replicas)
	}

	// replica status
	ic.Events = append(ic.Events,
		fmt.Sprintf("replicas: desired=%d ready=%d available=%d updated=%d",
			*deploy.Spec.Replicas,
			deploy.Status.ReadyReplicas,
			deploy.Status.AvailableReplicas,
			deploy.Status.UpdatedReplicas,
		),
	)

	// container info
	for _, c := range deploy.Spec.Template.Spec.Containers {
		ic.Identifiers["CONTAINER_NAME"] = c.Name
		ic.Identifiers["CURRENT_IMAGE"] = c.Image
		parts := strings.SplitN(c.Image, ":", 2)
		ic.Identifiers["IMAGE_BASE"] = parts[0]
		if len(parts) == 2 {
			ic.Identifiers["CURRENT_TAG"] = parts[1]
		} else {
			ic.Identifiers["CURRENT_TAG"] = "latest"
		}

		if c.Resources.Limits != nil {
			ic.Identifiers["CPU_LIMIT"] = c.Resources.Limits.Cpu().String()
			ic.Identifiers["MEMORY_LIMIT"] = c.Resources.Limits.Memory().String()
			ic.Events = append(ic.Events,
				fmt.Sprintf("container '%s' limits: cpu=%s memory=%s",
					c.Name,
					c.Resources.Limits.Cpu().String(),
					c.Resources.Limits.Memory().String(),
				),
			)
		} else {
			ic.Identifiers["CPU_LIMIT"] = "none"
			ic.Identifiers["MEMORY_LIMIT"] = "none"
			ic.Events = append(ic.Events,
				fmt.Sprintf("container '%s' limits: none", c.Name),
			)
		}

		if c.Resources.Requests != nil {
			ic.Identifiers["CPU_REQUEST"] = c.Resources.Requests.Cpu().String()
			ic.Identifiers["MEMORY_REQUEST"] = c.Resources.Requests.Memory().String()
			ic.Events = append(ic.Events,
				fmt.Sprintf("container '%s' requests: cpu=%s memory=%s",
					c.Name,
					c.Resources.Requests.Cpu().String(),
					c.Resources.Requests.Memory().String(),
				),
			)
		} else {
			ic.Identifiers["CPU_REQUEST"] = "none"
			ic.Identifiers["MEMORY_REQUEST"] = "none"
			ic.Events = append(ic.Events,
				fmt.Sprintf("container '%s' requests: none", c.Name),
			)
		}

		ic.Events = append(ic.Events,
			fmt.Sprintf("container '%s' image: %s", c.Name, c.Image),
		)
	}

	// always include pod states — LLM decides what matters
	b.enrichDeploymentPodStates(ctx, ic, snapshot)

	// warning events — max 10
	events, err := b.client.Kubernetes.CoreV1().Events(ic.ResourceNamespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s", ic.ResourceName),
	})
	if err == nil {
		count := 0
		for _, e := range events.Items {
			if e.Type == "Warning" && count < 10 {
				ic.Events = append(ic.Events,
					fmt.Sprintf("[Warning] %s: %s", e.Reason, e.Message),
				)
				count++
			}
		}
	}
}

// enrichDeploymentPodStates fetches pod states for a deployment
func (b *Builder) enrichDeploymentPodStates(ctx context.Context, ic *llm.IssueContext, snapshot *model.ClusterSnapshot) {
	for _, p := range snapshot.Pods {
		if p.Namespace != ic.ResourceNamespace || p.OwnerName != ic.ResourceName {
			continue
		}

		ic.Events = append(ic.Events,
			fmt.Sprintf("pod '%s' phase: %s", p.Name, p.Status),
		)

		for _, cs := range p.ContainerStates {
			if cs.WaitingReason != "" {
				ic.Events = append(ic.Events,
					fmt.Sprintf("  container '%s' waiting: %s — %s",
						cs.Name, cs.WaitingReason, cs.WaitingMessage),
				)
			}
			if cs.TerminatedReason != "" {
				ic.Events = append(ic.Events,
					fmt.Sprintf("  container '%s' terminated: %s exitCode: %d",
						cs.Name, cs.TerminatedReason, cs.ExitCode),
				)
			}
			ic.Events = append(ic.Events,
				fmt.Sprintf("  container '%s' restarts: %d ready: %v",
					cs.Name, cs.Restarts, cs.Ready),
			)
		}
	}
}

// enrichPodContext fetches focused context for a standalone pod
func (b *Builder) enrichPodContext(ctx context.Context, ic *llm.IssueContext) {
	pod, err := b.client.Kubernetes.CoreV1().
		Pods(ic.ResourceNamespace).
		Get(ctx, ic.ResourceName, metav1.GetOptions{})
	if err != nil {
		return
	}

	// standalone or managed
	if len(pod.OwnerReferences) == 0 {
		ic.Identifiers["POD_TYPE"] = "standalone"
		ic.Identifiers["FIX_METHOD"] = "delete and recreate with pinned image"
	} else {
		ic.Identifiers["POD_TYPE"] = "managed"
		owner := pod.OwnerReferences[0]
		if owner.Kind == "ReplicaSet" {
			parts := strings.Split(owner.Name, "-")
			if len(parts) > 1 {
				ic.Identifiers["OWNER_DEPLOYMENT"] = strings.Join(parts[:len(parts)-1], "-")
			}
		}
	}

	// container identifiers
	for _, c := range pod.Spec.Containers {
		ic.Identifiers["CONTAINER_NAME"] = c.Name
		ic.Identifiers["CURRENT_IMAGE"] = c.Image
		parts := strings.SplitN(c.Image, ":", 2)
		ic.Identifiers["IMAGE_BASE"] = parts[0]
		if len(parts) == 2 {
			ic.Identifiers["CURRENT_TAG"] = parts[1]
		} else {
			ic.Identifiers["CURRENT_TAG"] = "latest"
		}
		ic.Events = append(ic.Events,
			fmt.Sprintf("container '%s' image: %s", c.Name, c.Image),
		)

		if c.Resources.Limits != nil {
			ic.Identifiers["CPU_LIMIT"] = c.Resources.Limits.Cpu().String()
			ic.Identifiers["MEMORY_LIMIT"] = c.Resources.Limits.Memory().String()
		} else {
			ic.Identifiers["CPU_LIMIT"] = "none"
			ic.Identifiers["MEMORY_LIMIT"] = "none"
		}
	}

	// pod phase
	ic.Events = append(ic.Events,
		fmt.Sprintf("pod phase: %s", pod.Status.Phase),
	)

	// container statuses
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
			ic.Events = append(ic.Events,
				fmt.Sprintf("container '%s' waiting: %s — %s",
					cs.Name, cs.State.Waiting.Reason, cs.State.Waiting.Message),
			)
		}
		if cs.State.Terminated != nil {
			ic.Events = append(ic.Events,
				fmt.Sprintf("container '%s' terminated: %s exitCode: %d",
					cs.Name, cs.State.Terminated.Reason, cs.State.Terminated.ExitCode),
			)
		}
		if cs.LastTerminationState.Terminated != nil {
			ic.Events = append(ic.Events,
				fmt.Sprintf("container '%s' last termination: %s exitCode: %d",
					cs.Name,
					cs.LastTerminationState.Terminated.Reason,
					cs.LastTerminationState.Terminated.ExitCode),
			)
		}
		ic.Events = append(ic.Events,
			fmt.Sprintf("container '%s' restarts: %d ready: %v",
				cs.Name, cs.RestartCount, cs.Ready),
		)
	}

	// warning events max 10
	events, err := b.client.Kubernetes.CoreV1().Events(ic.ResourceNamespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s", ic.ResourceName),
	})
	if err == nil {
		count := 0
		for _, e := range events.Items {
			if e.Type == "Warning" && count < 10 {
				ic.Events = append(ic.Events,
					fmt.Sprintf("[Warning] %s: %s", e.Reason, e.Message),
				)
				count++
			}
		}
	}

	// current logs — last 20 lines
	tailLines := int64(20)
	logs, err := b.client.Kubernetes.CoreV1().
		Pods(ic.ResourceNamespace).
		GetLogs(ic.ResourceName, &corev1.PodLogOptions{
			TailLines: &tailLines,
		}).DoRaw(ctx)
	if err == nil && len(logs) > 0 {
		for _, l := range strings.Split(string(logs), "\n") {
			if l != "" {
				ic.Logs = append(ic.Logs, l)
			}
		}
	}

	// previous container logs
	prevLogs, err := b.client.Kubernetes.CoreV1().
		Pods(ic.ResourceNamespace).
		GetLogs(ic.ResourceName, &corev1.PodLogOptions{
			TailLines: &tailLines,
			Previous:  true,
		}).DoRaw(ctx)
	if err == nil && len(prevLogs) > 0 {
		ic.Logs = append(ic.Logs, "--- previous container logs ---")
		for _, l := range strings.Split(string(prevLogs), "\n") {
			if l != "" {
				ic.Logs = append(ic.Logs, l)
			}
		}
	}

	// node state
	if pod.Spec.NodeName != "" {
		ic.Identifiers["NODE_NAME"] = pod.Spec.NodeName
		node, err := b.client.Kubernetes.CoreV1().
			Nodes().Get(ctx, pod.Spec.NodeName, metav1.GetOptions{})
		if err == nil {
			var conditions []string
			for _, c := range node.Status.Conditions {
				conditions = append(conditions,
					fmt.Sprintf("%s=%s", c.Type, c.Status),
				)
			}
			ic.NodeState = fmt.Sprintf("node: %s  conditions: %s  cpu: %s  memory: %s",
				pod.Spec.NodeName,
				strings.Join(conditions, ", "),
				node.Status.Capacity.Cpu().String(),
				node.Status.Capacity.Memory().String(),
			)
		}
	}
}

// enrichNamespaceContext fetches focused context for a namespace
func (b *Builder) enrichNamespaceContext(ctx context.Context, ic *llm.IssueContext) {
	// existing network policies
	policies, err := b.client.Kubernetes.NetworkingV1().
		NetworkPolicies(ic.ResourceNamespace).
		List(ctx, metav1.ListOptions{})
	if err == nil {
		if len(policies.Items) == 0 {
			ic.Events = append(ic.Events,
				fmt.Sprintf("namespace '%s' has 0 network policies", ic.ResourceNamespace),
			)
		} else {
			for _, p := range policies.Items {
				ic.Events = append(ic.Events,
					fmt.Sprintf("existing network policy: %s", p.Name),
				)
			}
		}
	}

	// pod count
	pods, err := b.client.Kubernetes.CoreV1().
		Pods(ic.ResourceNamespace).
		List(ctx, metav1.ListOptions{})
	if err == nil {
		ic.Identifiers["POD_COUNT"] = fmt.Sprintf("%d", len(pods.Items))
		ic.Events = append(ic.Events,
			fmt.Sprintf("pods in namespace: %d", len(pods.Items)),
		)
	}

	// resource quotas
	quotas, err := b.client.Kubernetes.CoreV1().
		ResourceQuotas(ic.ResourceNamespace).
		List(ctx, metav1.ListOptions{})
	if err == nil && len(quotas.Items) > 0 {
		for _, q := range quotas.Items {
			ic.Events = append(ic.Events,
				fmt.Sprintf("resource quota: %s", q.Name),
			)
		}
	} else {
		ic.Events = append(ic.Events, "no resource quotas defined")
	}

	// deployments and their images — critical for CVE detection
	deployments, err := b.client.Kubernetes.AppsV1().
		Deployments(ic.ResourceNamespace).
		List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, d := range deployments.Items {
			for _, c := range d.Spec.Template.Spec.Containers {
				ic.Events = append(ic.Events,
					fmt.Sprintf("deployment '%s' container '%s' image: %s",
						d.Name, c.Name, c.Image),
				)
			}
		}
	}
}

// enrichIngressContext fetches context for an ingress
func (b *Builder) enrichIngressContext(ctx context.Context, ic *llm.IssueContext) {
	ing, err := b.client.Kubernetes.NetworkingV1().
		Ingresses(ic.ResourceNamespace).
		Get(ctx, ic.ResourceName, metav1.GetOptions{})
	if err != nil {
		return
	}

	if len(ing.Spec.TLS) == 0 {
		ic.Events = append(ic.Events, "ingress has no TLS configuration")
	} else {
		ic.Events = append(ic.Events,
			fmt.Sprintf("ingress has %d TLS configurations", len(ing.Spec.TLS)),
		)
	}

	for _, rule := range ing.Spec.Rules {
		ic.Identifiers["INGRESS_HOST"] = rule.Host
		ic.Events = append(ic.Events,
			fmt.Sprintf("host: %s", rule.Host),
		)
	}

	if ing.Spec.IngressClassName != nil {
		ic.Identifiers["INGRESS_CLASS"] = *ing.Spec.IngressClassName
		ic.Events = append(ic.Events,
			fmt.Sprintf("ingress class: %s", *ing.Spec.IngressClassName),
		)
	}
}

// enrichPVCContext fetches context for a PVC
func (b *Builder) enrichPVCContext(ctx context.Context, ic *llm.IssueContext) {
	pvc, err := b.client.Kubernetes.CoreV1().
		PersistentVolumeClaims(ic.ResourceNamespace).
		Get(ctx, ic.ResourceName, metav1.GetOptions{})
	if err != nil {
		return
	}

	ic.Identifiers["PVC_STATUS"] = string(pvc.Status.Phase)
	ic.Events = append(ic.Events,
		fmt.Sprintf("pvc status: %s", pvc.Status.Phase),
	)

	if pvc.Spec.StorageClassName != nil {
		ic.Identifiers["STORAGE_CLASS"] = *pvc.Spec.StorageClassName
		ic.Events = append(ic.Events,
			fmt.Sprintf("storage class: %s", *pvc.Spec.StorageClassName),
		)
	}

	if storage := pvc.Status.Capacity.Storage(); storage != nil {
		ic.Identifiers["CAPACITY"] = storage.String()
		ic.Events = append(ic.Events,
			fmt.Sprintf("capacity: %s", storage.String()),
		)
	}

	// check if mounted by any pod
	pods, err := b.client.Kubernetes.CoreV1().
		Pods(ic.ResourceNamespace).
		List(ctx, metav1.ListOptions{})
	if err == nil {
		mounted := false
		for _, pod := range pods.Items {
			for _, vol := range pod.Spec.Volumes {
				if vol.PersistentVolumeClaim != nil &&
					vol.PersistentVolumeClaim.ClaimName == ic.ResourceName {
					mounted = true
					ic.Events = append(ic.Events,
						fmt.Sprintf("mounted by pod: %s", pod.Name),
					)
				}
			}
		}
		if !mounted {
			ic.Events = append(ic.Events, "pvc is not mounted by any pod")
		}
	}
}

// enrichNodeContext fetches context for a node
func (b *Builder) enrichNodeContext(ctx context.Context, ic *llm.IssueContext) {
	node, err := b.client.Kubernetes.CoreV1().
		Nodes().Get(ctx, ic.ResourceName, metav1.GetOptions{})
	if err != nil {
		return
	}

	ic.Identifiers["NODE_NAME"] = ic.ResourceName

	for _, c := range node.Status.Conditions {
		ic.Events = append(ic.Events,
			fmt.Sprintf("condition %s=%s reason: %s message: %s",
				c.Type, c.Status, c.Reason, c.Message),
		)
	}

	ic.NodeState = fmt.Sprintf("cpu: %s  memory: %s  pods: %s",
		node.Status.Capacity.Cpu().String(),
		node.Status.Capacity.Memory().String(),
		node.Status.Capacity.Pods().String(),
	)

	// allocatable vs capacity
	ic.Events = append(ic.Events,
		fmt.Sprintf("allocatable cpu: %s  memory: %s",
			node.Status.Allocatable.Cpu().String(),
			node.Status.Allocatable.Memory().String(),
		),
	)

	// warning events max 10
	events, err := b.client.Kubernetes.CoreV1().Events("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s", ic.ResourceName),
	})
	if err == nil {
		count := 0
		for _, e := range events.Items {
			if e.Type == "Warning" && count < 10 {
				ic.Events = append(ic.Events,
					fmt.Sprintf("[Warning] %s: %s", e.Reason, e.Message),
				)
				count++
			}
		}
	}
}

// enrichServiceContext fetches context for a service
func (b *Builder) enrichServiceContext(ctx context.Context, ic *llm.IssueContext) {
	svc, err := b.client.Kubernetes.CoreV1().
		Services(ic.ResourceNamespace).
		Get(ctx, ic.ResourceName, metav1.GetOptions{})
	if err != nil {
		return
	}

	ic.Identifiers["SERVICE_TYPE"] = string(svc.Spec.Type)
	ic.Events = append(ic.Events,
		fmt.Sprintf("service type: %s", svc.Spec.Type),
	)

	// selector
	if len(svc.Spec.Selector) == 0 {
		ic.Events = append(ic.Events, "service has no selector — headless or external")
	} else {
		var selectors []string
		for k, v := range svc.Spec.Selector {
			selectors = append(selectors, fmt.Sprintf("%s=%s", k, v))
		}
		ic.Events = append(ic.Events,
			fmt.Sprintf("selector: %s", strings.Join(selectors, ", ")),
		)
	}

	// ports
	for _, p := range svc.Spec.Ports {
		ic.Events = append(ic.Events,
			fmt.Sprintf("port: %d → targetPort: %s protocol: %s",
				p.Port, p.TargetPort.String(), p.Protocol),
		)
	}

	// external IP for LoadBalancer
	if svc.Spec.Type == corev1.ServiceTypeLoadBalancer {
		if len(svc.Status.LoadBalancer.Ingress) == 0 {
			ic.Events = append(ic.Events, "loadbalancer has no external IP assigned")
		} else {
			ic.Events = append(ic.Events,
				fmt.Sprintf("external IP: %s", svc.Status.LoadBalancer.Ingress[0].IP),
			)
		}
	}

	// endpoints — most critical for detecting broken selectors
	endpoints, err := b.client.Kubernetes.CoreV1().
		Endpoints(ic.ResourceNamespace).
		Get(ctx, ic.ResourceName, metav1.GetOptions{})
	if err == nil {
		totalAddresses := 0
		for _, subset := range endpoints.Subsets {
			totalAddresses += len(subset.Addresses)
		}
		if totalAddresses == 0 {
			ic.Events = append(ic.Events, "endpoints: none — selector matches no pods")
		} else {
			ic.Events = append(ic.Events,
				fmt.Sprintf("endpoints: %d addresses ready", totalAddresses),
			)
		}
	}

	// warning events
	events, err := b.client.Kubernetes.CoreV1().Events(ic.ResourceNamespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s", ic.ResourceName),
	})
	if err == nil {
		count := 0
		for _, e := range events.Items {
			if e.Type == "Warning" && count < 10 {
				ic.Events = append(ic.Events,
					fmt.Sprintf("[Warning] %s: %s", e.Reason, e.Message),
				)
				count++
			}
		}
	}
}

// enrichGenericContext fetches warning events for any resource
func (b *Builder) enrichGenericContext(ctx context.Context, ic *llm.IssueContext) {
	events, err := b.client.Kubernetes.CoreV1().Events(ic.ResourceNamespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s", ic.ResourceName),
	})
	if err == nil {
		count := 0
		for _, e := range events.Items {
			if e.Type == "Warning" && count < 10 {
				ic.Events = append(ic.Events,
					fmt.Sprintf("[Warning] %s: %s", e.Reason, e.Message),
				)
				count++
			}
		}
	}
}
