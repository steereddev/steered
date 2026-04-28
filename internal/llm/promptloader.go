package llm

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// PromptLoader loads prompt files from ~/.steered/skills/
type PromptLoader struct {
	baseDir string
}

// NewPromptLoader creates a new PromptLoader
func NewPromptLoader() *PromptLoader {
	home, _ := os.UserHomeDir()
	return &PromptLoader{
		baseDir: filepath.Join(home, ".steered", "skills"),
	}
}

// LoadBase loads the base prompt from file or returns default
func (p *PromptLoader) LoadBase() string {
	return p.loadFile("base.md", defaultBasePrompt)
}

// LoadIssue loads resource-specific issue guidance from file
func (p *PromptLoader) LoadIssue(resourceKind string) string {
	filename := filepath.Join("resources", strings.ToLower(resourceKind)+".md")
	return p.loadFile(filename, p.defaultIssuePrompt(resourceKind))
}

func (p *PromptLoader) LoadSecurity() string {
	return p.loadFile(
		filepath.Join("security", "cve.md"),
		"",
	)
}

// loadFile reads a prompt file or returns fallback
func (p *PromptLoader) loadFile(name, fallback string) string {
	path := filepath.Join(p.baseDir, name)
	data, err := os.ReadFile(path)
	if err != nil {
		return fallback
	}
	return string(data)
}

// defaultIssuePrompt returns built-in default for each resource kind
func (p *PromptLoader) defaultIssuePrompt(kind string) string {
	switch strings.ToLower(kind) {
	case "deployment":
		return defaultDeploymentPrompt
	case "pod":
		return defaultPodPrompt
	case "namespace":
		return defaultNamespacePrompt
	case "node":
		return defaultNodePrompt
	case "ingress":
		return defaultIngressPrompt
	case "pvc":
		return defaultPVCPrompt
	default:
		return ""
	}
}

// InstallDefaultPrompts installs default prompt files on first run
func InstallDefaultPrompts() error {
	home, _ := os.UserHomeDir()
	promptDir := filepath.Join(home, ".steered", "skills", "resources")
	if err := os.MkdirAll(promptDir, 0755); err != nil {
		return err
	}

	files := map[string]string{
		"base.md":                 defaultBasePrompt,
		"resources/deployment.md": defaultDeploymentPrompt,
		"resources/pod.md":        defaultPodPrompt,
		"resources/namespace.md":  defaultNamespacePrompt,
		"resources/node.md":       defaultNodePrompt,
		"resources/ingress.md":    defaultIngressPrompt,
		"resources/pvc.md":        defaultPVCPrompt,
	}

	for name, content := range files {
		path := filepath.Join(home, ".steered", "skills", name)
		// never overwrite user edits
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

// UpdateSkills fetches latest skills from steereddev/steered-skills
func UpdateSkills() error {
	baseURL := "https://raw.githubusercontent.com/steereddev/steered-skills/main"

	files := []string{
		"skills/base.md",
		"skills/resources/deployment.md",
		"skills/resources/pod.md",
		"skills/resources/namespace.md",
		"skills/resources/node.md",
		"skills/resources/ingress.md",
		"skills/resources/pvc.md",
		"skills/security/cve.md",
	}

	home, _ := os.UserHomeDir()
	skillsDir := filepath.Join(home, ".steered", "skills")

	fmt.Println("updating steered skills...")
	fmt.Println()

	updated := 0
	for _, file := range files {
		url := fmt.Sprintf("%s/%s", baseURL, file)

		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("  ⚠  skipping %s — %v\n", file, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("  ⚠  skipping %s — not found\n", file)
			continue
		}

		content, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("  ⚠  skipping %s — %v\n", file, err)
			continue
		}

		localPath := filepath.Join(skillsDir,
			strings.TrimPrefix(file, "skills/"))

		if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
			fmt.Printf("  ⚠  skipping %s — %v\n", file, err)
			continue
		}

		if err := os.WriteFile(localPath, content, 0644); err != nil {
			fmt.Printf("  ⚠  skipping %s — %v\n", file, err)
			continue
		}

		fmt.Printf("  ✓  %s\n", file)
		updated++
	}

	fmt.Println()
	fmt.Printf("✓ %d skills updated\n", updated)
	fmt.Println("  restart steered to apply new skills")
	return nil
}

// SkillsExist checks if skills have been installed
func SkillsExist() bool {
	home, _ := os.UserHomeDir()
	basePath := filepath.Join(home, ".steered", "skills", "base.md")
	_, err := os.Stat(basePath)
	return err == nil
}
