package llm

import (
	"fmt"
	"strings"
)

// BuildDetectPrompt builds a prompt that asks LLM to find AND explain all issues
func BuildDetectPrompt(ic *IssueContext) string {
	loader := NewPromptLoader()
	var b strings.Builder

	b.WriteString(loader.LoadBase())
	b.WriteString("\n\n")

	b.WriteString("## THE RESOURCE TO ANALYZE\n")
	b.WriteString(buildResourceDescription(ic))
	b.WriteString("\n")

	// policy context from skills files
	if ic.PolicyContext != "" {
		b.WriteString("## POLICY AND BEST PRACTICES\n")
		b.WriteString(ic.PolicyContext)
		b.WriteString("\n")
	}

	// events
	if len(ic.Events) > 0 {
		b.WriteString("\n## CURRENT STATE\n")
		for _, e := range ic.Events {
			b.WriteString(fmt.Sprintf("- %s\n", e))
		}
	}

	// node state
	if ic.NodeState != "" {
		b.WriteString(fmt.Sprintf("\n## NODE STATE\n%s\n", ic.NodeState))
	}

	// logs
	if len(ic.Logs) > 0 {
		b.WriteString("\n## CONTAINER LOGS\n")
		for _, l := range ic.Logs {
			b.WriteString(fmt.Sprintf("  %s\n", l))
		}
	}

	b.WriteString(`
## YOUR TASK

Review the resource state above and identify ALL issues.
For each issue provide a complete analysis and fix.

Respond ONLY with a JSON array:
[
  {
    "severity": "critical|warning|security",
    "resource_type": "deployment|pod|namespace|node|pvc|ingress",
    "title": "short description of the issue",
    "resource": "exact resource name",
    "namespace": "exact namespace",
    "meta": "one line supporting detail",
    "root_cause": "max 15 words — why this is happening",
    "fix_explanation": "max 15 words — what needs to be done",
    "command": "best kubectl command to fix or investigate",
    "watch_for": "single kubectl command to confirm fix",
    "risk": "max 15 words — consequence if not fixed",
    "confidence": "high|medium|low",
    "type": "fix|investigate"
  }
]

RULES:
- only report real issues visible in the state above
- use ONLY exact names and values from THE RESOURCE TO ANALYZE
- never invent values from training knowledge
- if no issues found return empty array: []
- severity critical → application impact
- severity warning  → best practice violation
- severity security → security misconfiguration
- type fix         → command directly resolves issue
- type investigate → needs manual investigation first
- never use :latest as replacement — use <valid-tag>
- never use --grace-period=0 or --force
- multi-line commands allowed — contractor copies via clipboard

respond with JSON array only — no markdown, no explanation
`)

	return b.String()
}

// buildResourceDescription builds natural language resource summary
func buildResourceDescription(ic *IssueContext) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("- Resource type: %s\n", ic.ResourceKind))
	b.WriteString(fmt.Sprintf("- Name: %s\n", ic.ResourceName))
	b.WriteString(fmt.Sprintf("- Namespace: %s\n", ic.ResourceNamespace))

	if v, ok := ic.Identifiers["CONTAINER_NAME"]; ok {
		b.WriteString(fmt.Sprintf("- Container name: %s\n", v))
	}
	if v, ok := ic.Identifiers["CURRENT_IMAGE"]; ok {
		b.WriteString(fmt.Sprintf("- Current image: %s\n", v))
	}
	if v, ok := ic.Identifiers["IMAGE_BASE"]; ok {
		b.WriteString(fmt.Sprintf("- Image base: %s\n", v))
	}
	if v, ok := ic.Identifiers["CURRENT_TAG"]; ok {
		b.WriteString(fmt.Sprintf("- Current tag: %s\n", v))
	}
	if v, ok := ic.Identifiers["CPU_LIMIT"]; ok && ic.Identifiers["CPU_LIMIT"] != "" {
		b.WriteString(fmt.Sprintf("- CPU limit: %s\n", v))
	} else {
		b.WriteString("- CPU limit: not set\n")
	}
	if v, ok := ic.Identifiers["MEMORY_LIMIT"]; ok && ic.Identifiers["MEMORY_LIMIT"] != "" {
		b.WriteString(fmt.Sprintf("- Memory limit: %s\n", v))
	} else {
		b.WriteString("- Memory limit: not set\n")
	}
	if v, ok := ic.Identifiers["CPU_REQUEST"]; ok {
		b.WriteString(fmt.Sprintf("- CPU request: %s\n", v))
	}
	if v, ok := ic.Identifiers["MEMORY_REQUEST"]; ok {
		b.WriteString(fmt.Sprintf("- Memory request: %s\n", v))
	}
	if v, ok := ic.Identifiers["REPLICA_COUNT"]; ok {
		b.WriteString(fmt.Sprintf("- Replicas: %s\n", v))
	}
	if v, ok := ic.Identifiers["POD_TYPE"]; ok {
		b.WriteString(fmt.Sprintf("- Pod type: %s\n", v))
	}
	if v, ok := ic.Identifiers["OWNER_DEPLOYMENT"]; ok {
		b.WriteString(fmt.Sprintf("- Owner deployment: %s\n", v))
	}
	if v, ok := ic.Identifiers["POD_COUNT"]; ok {
		b.WriteString(fmt.Sprintf("- Pods in namespace: %s\n", v))
	}
	if v, ok := ic.Identifiers["NODE_NAME"]; ok {
		b.WriteString(fmt.Sprintf("- Node: %s\n", v))
	}
	if v, ok := ic.Identifiers["INGRESS_HOST"]; ok {
		b.WriteString(fmt.Sprintf("- Ingress host: %s\n", v))
	}
	if v, ok := ic.Identifiers["INGRESS_CLASS"]; ok {
		b.WriteString(fmt.Sprintf("- Ingress class: %s\n", v))
	}
	if v, ok := ic.Identifiers["STORAGE_CLASS"]; ok {
		b.WriteString(fmt.Sprintf("- Storage class: %s\n", v))
	}
	if v, ok := ic.Identifiers["CAPACITY"]; ok {
		b.WriteString(fmt.Sprintf("- Capacity: %s\n", v))
	}

	return b.String()
}
