package collector

import (
	"context"
	"fmt"
	"strings"

	"github.com/steereddev/steered/internal/client"
	"github.com/steereddev/steered/internal/model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceCollector collects service information from the cluster
type ServiceCollector struct {
	client *client.Client
}

// NewServiceCollector creates a new ServiceCollector
func NewServiceCollector(c *client.Client) *ServiceCollector {
	return &ServiceCollector{client: c}
}

// Collect gathers all service data and fills snapshot.Services
func (s *ServiceCollector) Collect(ctx context.Context, snapshot *model.ClusterSnapshot) error {
	services, err := s.client.Kubernetes.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to collect services: %w", err)
	}

	for _, svc := range services.Items {
		// build ports string
		var ports []string
		for _, p := range svc.Spec.Ports {
			if p.NodePort != 0 {
				ports = append(ports, fmt.Sprintf("%d:%d/%s", p.Port, p.NodePort, p.Protocol))
			} else {
				ports = append(ports, fmt.Sprintf("%d/%s", p.Port, p.Protocol))
			}
		}

		// get external ip
		externalIP := ""
		switch svc.Spec.Type {
		case corev1.ServiceTypeLoadBalancer:
			for _, ing := range svc.Status.LoadBalancer.Ingress {
				if ing.Hostname != "" {
					externalIP = ing.Hostname
				} else if ing.IP != "" {
					externalIP = ing.IP
				}
			}
			if externalIP == "" {
				externalIP = "<pending>"
			}
		case corev1.ServiceTypeNodePort:
			externalIP = "<nodes>"
		case corev1.ServiceTypeExternalName:
			externalIP = svc.Spec.ExternalName
		}

		snapshot.Services = append(snapshot.Services, model.Service{
			Name:       svc.Name,
			Namespace:  svc.Namespace,
			Type:       string(svc.Spec.Type),
			ClusterIP:  svc.Spec.ClusterIP,
			ExternalIP: externalIP,
			Ports:      strings.Join(ports, ", "),
			Age:        age(svc.CreationTimestamp.Time),
		})
	}

	return nil
}
