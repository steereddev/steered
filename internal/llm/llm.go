package llm

import "context"

// Analyzer defines the interface all LLM providers must implement
type Analyzer interface {
	Analyze(ctx context.Context, ic *IssueContext) ([]*Issue, error)
	Name() string
}

// IssueContext is the rich context sent to the LLM
type IssueContext struct {
	ResourceName      string
	ResourceNamespace string
	ResourceKind      string
	Identifiers       map[string]string
	Events            []string
	Logs              []string
	NodeState         string
	ClusterName       string
	PolicyContext     string // from skills files
}

// Issue is what LLM returns — detected + explained
type Issue struct {
	Severity       string `json:"severity"`
	ResourceType   string `json:"resource_type"`
	Title          string `json:"title"`
	Resource       string `json:"resource"`
	Namespace      string `json:"namespace"`
	Meta           string `json:"meta"`
	RootCause      string `json:"root_cause"`
	FixExplanation string `json:"fix_explanation"`
	Command        string `json:"command"`
	WatchFor       string `json:"watch_for"`
	Risk           string `json:"risk"`
	Confidence     string `json:"confidence"`
	Type           string `json:"type"`
}
