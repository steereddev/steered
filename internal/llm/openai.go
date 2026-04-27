package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// OpenAIAnalyzer implements Analyzer for OpenAI
type OpenAIAnalyzer struct {
	endpoint string
	model    string
	apiKey   string
}

// NewOpenAIAnalyzer creates a new OpenAIAnalyzer
func NewOpenAIAnalyzer(endpoint, model, apiKey string) *OpenAIAnalyzer {
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1"
	}
	if model == "" {
		model = "gpt-4o"
	}
	return &OpenAIAnalyzer{endpoint: endpoint, model: model, apiKey: apiKey}
}

// Name returns the provider name
func (o *OpenAIAnalyzer) Name() string {
	return fmt.Sprintf("openai/%s", o.model)
}

// Analyze detects and explains all issues for a resource
func (o *OpenAIAnalyzer) Analyze(ctx context.Context, ic *IssueContext) ([]*Issue, error) {
	prompt := BuildDetectPrompt(ic)
	DebugPrompt(prompt)

	reqBody := map[string]interface{}{
		"model": o.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"response_format": map[string]string{"type": "json_object"},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		o.endpoint+"/chat/completions",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var openaiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &openaiResp); err != nil {
		return nil, fmt.Errorf("failed to parse openai response: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	DebugResponse(openaiResp.Choices[0].Message.Content)
	return parseIssues(openaiResp.Choices[0].Message.Content)
}
