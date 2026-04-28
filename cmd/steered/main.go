package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/steereddev/steered/internal/client"
	"github.com/steereddev/steered/internal/config"
	"github.com/steereddev/steered/internal/llm"
	"github.com/steereddev/steered/internal/tui"
)

func main() {
	// init config manager
	cfgManager, err := config.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init config: %v\n", err)
		os.Exit(1)
	}

	// fetch skills on first install only
	if !llm.SkillsExist() {
		fmt.Println("first run — fetching latest skills from steereddev/steered-skills...")
		if err := llm.UpdateSkills(); err != nil {
			fmt.Println("could not fetch skills — run: steered --update-skills")
			fmt.Println("continuing with minimal defaults...")
		}
	}

	// check for --setup flag or --clear flag
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--setup":
			if _, err := config.RunSetup(cfgManager); err != nil {
				fmt.Fprintf(os.Stderr, "setup failed: %v\n", err)
				os.Exit(1)
			}
			return
		case "--clear":
			if err := cfgManager.ClearAll(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to clear config: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("config cleared")
			return
		case "--update-skills":
			if err := llm.UpdateSkills(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to update skills: %v\n", err)
				os.Exit(1)
			}
			return
		case "--debug":
			llm.EnableDebug()
		}
	}

	// LLM is required for steered to work
	if !cfgManager.IsLLMConfigured() {
		fmt.Println("steered requires an LLM to analyze your cluster.")
		fmt.Println("let's configure one now.") //yels
		result, err := config.RunSetup(cfgManager)
		if err != nil {
			fmt.Fprintf(os.Stderr, "setup failed: %v\n", err)
			os.Exit(1)
		}
		_ = result
	}

	// build k8s client
	kubeconfigPath := ""
	kubecontext := ""
	for i, arg := range os.Args[1:] {
		if arg == "--kubeconfig" && i+1 < len(os.Args[1:]) {
			kubeconfigPath = os.Args[i+2]
		}
		if arg == "--context" && i+1 < len(os.Args[1:]) {
			kubecontext = os.Args[i+2]
		}
	}

	c, err := client.New(kubeconfigPath, kubecontext)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to cluster: %v\n", err)
		os.Exit(1)
	}

	// start bubbletea live TUI
	m := tui.New(c, cfgManager)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start steered: %v\n", err)
		os.Exit(1)
	}
}
