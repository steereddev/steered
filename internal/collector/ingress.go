package collector

import (
	"context"
	"fmt"
	"strings"

	"github.com/steereddev/steered/internal/client"
	"github.com/steereddev/steered/internal/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IngressCollector collects ingress information from the cluster
type IngressCollector struct {
	client *client.Client
}

// NewIngressCollector creates a new IngressCollector
func NewIngressCollector(c *client.Client) *IngressCollector {
	return &IngressCollector{client: c}
}

// Collect gathers all ingress data and fills snapshot.Ingresses
func (i *IngressCollector) Collect(ctx context.Context, snapshot *model.ClusterSnapshot) error {
	ingresses, err := i.client.Kubernetes.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to collect ingresses: %w", err)
	}

	for _, ing := range ingresses.Items {
		// get ingress class
		class := ""
		if ing.Spec.IngressClassName != nil {
			class = *ing.Spec.IngressClassName
		}

		// collect all hosts
		var hosts []string
		for _, rule := range ing.Spec.Rules {
			if rule.Host != "" {
				hosts = append(hosts, rule.Host)
			}
		}

		// get address
		var addresses []string
		for _, lb := range ing.Status.LoadBalancer.Ingress {
			if lb.Hostname != "" {
				addresses = append(addresses, lb.Hostname)
			} else if lb.IP != "" {
				addresses = append(addresses, lb.IP)
			}
		}

		// collect ports
		ports := "80"
		for _, tls := range ing.Spec.TLS {
			if len(tls.Hosts) > 0 {
				ports = "80, 443"
				break
			}
		}

		snapshot.Ingresses = append(snapshot.Ingresses, model.Ingress{
			Name:      ing.Name,
			Namespace: ing.Namespace,
			Class:     class,
			Hosts:     strings.Join(hosts, ", "),
			Address:   strings.Join(addresses, ", "),
			Ports:     ports,
			Age:       age(ing.CreationTimestamp.Time),
		})
	}

	return nil
}
