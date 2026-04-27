package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// OllamaAnalyzer implements Analyzer for local ollama
type OllamaAnalyzer struct {
	endpoint string
	model    string
}

// NewOllamaAnalyzer creates a new OllamaAnalyzer
func NewOllamaAnalyzer(endpoint, model string) *OllamaAnalyzer {
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	if model == "" {
		model = "gemma3:4b"
	}
	return &OllamaAnalyzer{endpoint: endpoint, model: model}
}

// Name returns the provider name
func (o *OllamaAnalyzer) Name() string {
	return fmt.Sprintf("ollama/%s", o.model)
}

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaResponse struct {
	Response string `json:"response"`
}

// Analyze detects and explains all issues for a resource
func (o *OllamaAnalyzer) Analyze(ctx context.Context, ic *IssueContext) ([]*Issue, error) {
	prompt := BuildDetectPrompt(ic)
	DebugPrompt(prompt)

	reqBody := ollamaRequest{
		Model:  o.model,
		Prompt: prompt,
		Stream: false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		o.endpoint+"/api/generate",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var ollamaResp ollamaResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to parse ollama response: %w", err)
	}

	DebugResponse(ollamaResp.Response)
	return parseIssues(ollamaResp.Response)
}

// parseIssues parses LLM response as array of issues
func parseIssues(response string) ([]*Issue, error) {
	clean := strings.TrimSpace(response)
	clean = strings.TrimPrefix(clean, "```json")
	clean = strings.TrimPrefix(clean, "```")
	clean = strings.TrimSuffix(clean, "```")
	clean = strings.TrimSpace(clean)

	start := strings.Index(clean, "[")
	end := strings.LastIndex(clean, "]")
	if start == -1 || end == -1 {
		// empty or no issues
		return []*Issue{}, nil
	}

	clean = clean[start : end+1]
	var issues []*Issue
	if err := json.Unmarshal([]byte(clean), &issues); err != nil {
		return nil, fmt.Errorf("failed to parse issues: %w", err)
	}

	return issues, nil
}
