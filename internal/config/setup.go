package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

// Provider constants
const (
	ProviderOllama    = "ollama"
	ProviderOpenAI    = "openai"
	ProviderAnthropic = "anthropic"
	ProviderSkip      = "skip"
)

// ProviderDefaults holds default model and endpoint per provider
var ProviderDefaults = map[string]struct {
	Model    string
	Endpoint string
}{
	ProviderOllama:    {Model: "llama3", Endpoint: "http://localhost:11434"},
	ProviderOpenAI:    {Model: "gpt-4o", Endpoint: "https://api.openai.com/v1"},
	ProviderAnthropic: {Model: "claude-3-5-sonnet-20241022", Endpoint: "https://api.anthropic.com"},
}

// SetupResult holds the result of the setup flow
type SetupResult struct {
	Provider string
	Model    string
	Endpoint string
	APIKey   string
	Skipped  bool
}

// RunSetup runs the interactive first run setup
func RunSetup(m *Manager) (*SetupResult, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("  ▸  S T E E R E D  —  LLM Setup")
	fmt.Println()
	fmt.Println("  configure a language model for live cluster analysis")
	fmt.Println("  your API key is stored locally with 24h expiry")
	fmt.Println()
	fmt.Println("  [1]  ollama      local, free, private, recommended")
	fmt.Println("  [2]  anthropic   best reasoning, requires API key")
	fmt.Println("  [3]  openai      best quality, requires API key")
	fmt.Println("  [4]  skip        snapshot only, configure later")
	fmt.Println()
	fmt.Print("  select provider [1-4]: ")

	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}
	input = strings.TrimSpace(input)

	var result SetupResult

	switch input {
	case "1":
		result.Provider = ProviderOllama
		result.Model = ProviderDefaults[ProviderOllama].Model
		result.Endpoint = ProviderDefaults[ProviderOllama].Endpoint
		result.APIKey = ""

		fmt.Println()
		fmt.Println("  ollama selected")
		fmt.Printf("  make sure ollama is running: ollama serve\n")
		fmt.Printf("  make sure model is pulled:   ollama pull %s\n", result.Model)

		// ask for custom model
		fmt.Printf("  model [%s]: ", result.Model)
		modelInput, _ := reader.ReadString('\n')
		modelInput = strings.TrimSpace(modelInput)
		if modelInput != "" {
			result.Model = modelInput
		}

	case "2":
		result.Provider = ProviderAnthropic
		result.Model = ProviderDefaults[ProviderAnthropic].Model
		result.Endpoint = ProviderDefaults[ProviderAnthropic].Endpoint

		fmt.Println()
		fmt.Print("  enter Anthropic API key: ")
		keyInput, _ := reader.ReadString('\n')
		result.APIKey = strings.TrimSpace(keyInput)

		if result.APIKey == "" {
			fmt.Println("  no API key provided, skipping")
			result.Skipped = true
			return &result, nil
		}

		fmt.Printf("  model [%s]: ", result.Model)
		modelInput, _ := reader.ReadString('\n')
		modelInput = strings.TrimSpace(modelInput)
		if modelInput != "" {
			result.Model = modelInput
		}

	case "3":
		result.Provider = ProviderOpenAI
		result.Model = ProviderDefaults[ProviderOpenAI].Model
		result.Endpoint = ProviderDefaults[ProviderOpenAI].Endpoint

		fmt.Println()
		fmt.Print("  enter OpenAI API key: ")
		keyInput, _ := reader.ReadString('\n')
		result.APIKey = strings.TrimSpace(keyInput)

		if result.APIKey == "" {
			fmt.Println("  no API key provided, skipping")
			result.Skipped = true
			return &result, nil
		}

		fmt.Printf("  model [%s]: ", result.Model)
		modelInput, _ := reader.ReadString('\n')
		modelInput = strings.TrimSpace(modelInput)
		if modelInput != "" {
			result.Model = modelInput
		}

	default:
		result.Skipped = true
		fmt.Println()
		fmt.Println("  skipping LLM setup — run steered --setup to configure later")
		return &result, nil
	}

	// save config
	cfg := &Config{
		LLMProvider:   result.Provider,
		LLMModel:      result.Model,
		LLMEndpoint:   result.Endpoint,
		ProbeInterval: 30,
	}

	if err := m.SaveConfig(cfg); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	// save API key with TTL if provided
	if result.APIKey != "" {
		if err := m.SaveAPIKey(result.APIKey, DefaultKeyTTL); err != nil {
			return nil, fmt.Errorf("failed to save API key: %w", err)
		}
		fmt.Printf("\n  API key saved with 24h expiry\n")
	}

	fmt.Printf("\n  ✓ configured: %s / %s\n", result.Provider, result.Model)
	fmt.Println("  starting steered...")
	fmt.Println()

	return &result, nil
}

// RenewAPIKey extends the TTL of the current API key
func RenewAPIKey(m *Manager, ttl time.Duration) error {
	key, err := m.LoadAPIKey()
	if err != nil {
		return err
	}
	if key == "" {
		return fmt.Errorf("no API key found — run steered --setup")
	}
	return m.SaveAPIKey(key, ttl)
}
