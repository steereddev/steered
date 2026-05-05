package tui

import (
	"crypto/md5"
	"fmt"
	"strings"
	"time"

	"github.com/steereddev/steered/internal/model"
)

// Issue represents a detected cluster issue with full analysis
type Issue struct {
	Severity       string
	ResourceType   string
	Title          string
	Resource       string
	Namespace      string
	Meta           string
	RootCause      string
	FixExplanation string
	Command        string
	WatchFor       string
	Risk           string
	Confidence     string
	Type           string
	DetectedAt     time.Time
}

// resourceKey generates a unique key for a resource
func resourceKey(kind, name, namespace string) string {
	return kind + "/" + name + "/" + namespace
}

// resourceHash generates a hash of resource state for change detection
func resourceHash(events []string) string {
	h := md5.New()
	for _, e := range events {
		h.Write([]byte(e))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// buildResourceList returns all resources that need analysis
func buildResourceList(snapshot *model.ClusterSnapshot) []resourceToAnalyze {
	var resources []resourceToAnalyze

	// deployments
	for _, d := range snapshot.Deployments {
		resources = append(resources, resourceToAnalyze{
			Kind:      "deployment",
			Name:      d.Name,
			Namespace: d.Namespace,
		})
	}

	// standalone pods
	for _, p := range snapshot.Pods {
		if p.OwnerKind == "" || p.OwnerName == "" {
			resources = append(resources, resourceToAnalyze{
				Kind:      "pod",
				Name:      p.Name,
				Namespace: p.Namespace,
			})
		}
	}

	// namespaces
	for _, ns := range snapshot.Namespaces {
		resources = append(resources, resourceToAnalyze{
			Kind:      "namespace",
			Name:      ns.Name,
			Namespace: ns.Name,
		})
	}

	// nodes
	for _, n := range snapshot.Nodes {
		resources = append(resources, resourceToAnalyze{
			Kind:      "node",
			Name:      n.Name,
			Namespace: "cluster",
		})
	}

	// pvcs
	for _, pvc := range snapshot.PVCs {
		resources = append(resources, resourceToAnalyze{
			Kind:      "pvc",
			Name:      pvc.Name,
			Namespace: pvc.Namespace,
		})
	}

	// ingresses
	for _, ing := range snapshot.Ingresses {
		resources = append(resources, resourceToAnalyze{
			Kind:      "ingress",
			Name:      ing.Name,
			Namespace: ing.Namespace,
		})
	}

	return resources
}

type resourceToAnalyze struct {
	Kind      string
	Name      string
	Namespace string
}

// issueKey generates unique key for an issue
func issueKey(issue Issue) string {
	return issue.ResourceType + issue.Resource + issue.Title
}

// filterBySeverity returns issues of specific severity
func filterBySeverity(issues []Issue, severity string) []Issue {
	var filtered []Issue
	for _, i := range issues {
		if i.Severity == severity {
			filtered = append(filtered, i)
		}
	}
	return filtered
}

// topIssues returns top N issues by priority
func topIssues(issues []Issue, n int) []Issue {
	var critical, security, warning []Issue
	for _, i := range issues {
		switch i.Severity {
		case "critical":
			critical = append(critical, i)
		case "security":
			security = append(security, i)
		default:
			warning = append(warning, i)
		}
	}

	var result []Issue
	result = append(result, critical...)
	result = append(result, security...)
	result = append(result, warning...)

	if len(result) > n {
		return result[:n]
	}
	return result
}

// findNewIssues returns issues not seen in previous probe
func findNewIssues(current, previous []Issue) []Issue {
	prevKeys := make(map[string]bool)
	for _, p := range previous {
		prevKeys[issueKey(p)] = true
	}

	var newIssues []Issue
	for _, c := range current {
		if !prevKeys[issueKey(c)] {
			newIssues = append(newIssues, c)
		}
	}
	return newIssues
}

// systemNamespaces returns true for kube system namespaces
func isSystemNamespace(ns string) bool {
	system := map[string]bool{
		"kube-system":     true,
		"kube-public":     true,
		"kube-node-lease": true,
	}
	return system[ns]
}

// truncate truncates a string to max length
func truncate(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// removeStaleIssues removes issues for resources no longer in snapshot
func removeStaleIssues(issues []Issue, snapshot *model.ClusterSnapshot) []Issue {
	// build set of existing resources
	existing := make(map[string]bool)

	for _, d := range snapshot.Deployments {
		existing[resourceKey("deployment", d.Name, d.Namespace)] = true
	}
	for _, p := range snapshot.Pods {
		if p.OwnerKind == "" {
			existing[resourceKey("pod", p.Name, p.Namespace)] = true
		}
	}
	for _, n := range snapshot.Namespaces {
		existing[resourceKey("namespace", n.Name, n.Name)] = true
	}
	for _, node := range snapshot.Nodes {
		existing[resourceKey("node", node.Name, "cluster")] = true
	}
	for _, pvc := range snapshot.PVCs {
		existing[resourceKey("pvc", pvc.Name, pvc.Namespace)] = true
	}
	for _, ing := range snapshot.Ingresses {
		existing[resourceKey("ingress", ing.Name, ing.Namespace)] = true
	}

	var filtered []Issue
	for _, issue := range issues {
		key := resourceKey(issue.ResourceType, issue.Resource, issue.Namespace)
		if existing[key] {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

func severeIssues(issues []Issue) int {
	count := 0
	for _, i := range issues {
		if i.Severity == "critical" || i.Severity == "security" {
			count++
		}
	}
	return count
}
