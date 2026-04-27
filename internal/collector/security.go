package collector

import (
	"context"
	"fmt"
	"strings"

	"github.com/steereddev/steered/internal/client"
	"github.com/steereddev/steered/internal/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecurityCollector collects security signals from the cluster
type SecurityCollector struct {
	client *client.Client
}

// NewSecurityCollector creates a new SecurityCollector
func NewSecurityCollector(c *client.Client) *SecurityCollector {
	return &SecurityCollector{client: c}
}

// Collect gathers security signals and fills snapshot.SecuritySignals
func (s *SecurityCollector) Collect(ctx context.Context, snapshot *model.ClusterSnapshot) error {
	if err := s.collectPodSecuritySignals(ctx, snapshot); err != nil {
		return err
	}
	if err := s.collectIngressTLSSignals(ctx, snapshot); err != nil {
		return err
	}
	if err := s.collectNetworkPolicySignals(ctx, snapshot); err != nil {
		return err
	}
	return nil
}

// collectPodSecuritySignals checks pods for security misconfigurations
func (s *SecurityCollector) collectPodSecuritySignals(ctx context.Context, snapshot *model.ClusterSnapshot) error {
	pods, err := s.client.Kubernetes.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to collect pods for security check: %w", err)
	}

	for _, pod := range pods.Items {
		ref := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)

		// detect owner
		ownerRef := "standalone"
		if len(pod.OwnerReferences) > 0 {
			owner := pod.OwnerReferences[0]
			if owner.Kind == "ReplicaSet" {
				parts := strings.Split(owner.Name, "-")
				if len(parts) > 1 {
					ownerRef = "deployment:" + strings.Join(parts[:len(parts)-1], "-")
				}
			} else {
				ownerRef = strings.ToLower(owner.Kind) + ":" + owner.Name
			}
		}

		// check host network
		if pod.Spec.HostNetwork {
			snapshot.SecuritySignals.HostNetworkPods = append(
				snapshot.SecuritySignals.HostNetworkPods, ref,
			)
		}

		for _, c := range pod.Spec.Containers {
			containerRef := fmt.Sprintf("%s/%s/%s", ref, c.Name, ownerRef)

			// check privileged containers
			if c.SecurityContext != nil && c.SecurityContext.Privileged != nil && *c.SecurityContext.Privileged {
				snapshot.SecuritySignals.PrivilegedContainers = append(
					snapshot.SecuritySignals.PrivilegedContainers, containerRef,
				)
			}

			// check containers running as root
			if c.SecurityContext != nil && c.SecurityContext.RunAsUser != nil && *c.SecurityContext.RunAsUser == 0 {
				snapshot.SecuritySignals.ContainersRunningAsRoot = append(
					snapshot.SecuritySignals.ContainersRunningAsRoot, containerRef,
				)
			}

			// check no security context
			if c.SecurityContext == nil {
				snapshot.SecuritySignals.NoSecurityContext = append(
					snapshot.SecuritySignals.NoSecurityContext, containerRef,
				)
			}

			// check latest image tag
			if strings.HasSuffix(c.Image, ":latest") || !strings.Contains(c.Image, ":") {
				snapshot.SecuritySignals.LatestImageTags = append(
					snapshot.SecuritySignals.LatestImageTags,
					fmt.Sprintf("%s/%s/%s/%s", pod.Namespace, pod.Name, c.Name, ownerRef),
				)
			}

			// check secrets in env vars
			for _, env := range c.Env {
				name := strings.ToLower(env.Name)
				if strings.Contains(name, "secret") ||
					strings.Contains(name, "password") ||
					strings.Contains(name, "token") ||
					strings.Contains(name, "key") ||
					strings.Contains(name, "api_key") {
					if env.Value != "" {
						snapshot.SecuritySignals.SecretsInEnvVars = append(
							snapshot.SecuritySignals.SecretsInEnvVars, containerRef,
						)
						break
					}
				}
			}
		}
	}

	return nil
}

// collectIngressTLSSignals checks ingresses for missing TLS
func (s *SecurityCollector) collectIngressTLSSignals(ctx context.Context, snapshot *model.ClusterSnapshot) error {
	ingresses, err := s.client.Kubernetes.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to collect ingresses for security check: %w", err)
	}

	for _, ing := range ingresses.Items {
		if len(ing.Spec.TLS) == 0 {
			snapshot.SecuritySignals.IngressesWithoutTLS = append(
				snapshot.SecuritySignals.IngressesWithoutTLS,
				fmt.Sprintf("%s/%s", ing.Namespace, ing.Name),
			)
		}
	}

	return nil
}

// collectNetworkPolicySignals checks namespaces for missing network policies
func (s *SecurityCollector) collectNetworkPolicySignals(ctx context.Context, snapshot *model.ClusterSnapshot) error {
	namespaces, err := s.client.Kubernetes.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to collect namespaces for security check: %w", err)
	}

	systemNamespaces := map[string]bool{
		"kube-system":     true,
		"kube-public":     true,
		"kube-node-lease": true,
	}

	for _, ns := range namespaces.Items {
		if systemNamespaces[ns.Name] {
			continue
		}

		policies, err := s.client.Kubernetes.NetworkingV1().NetworkPolicies(ns.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			continue
		}

		if len(policies.Items) == 0 {
			snapshot.SecuritySignals.NamespacesWithoutNetPol = append(
				snapshot.SecuritySignals.NamespacesWithoutNetPol, ns.Name,
			)
		}
	}

	return nil
}
