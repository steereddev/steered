package tui

import (
	"context"
	"crypto/md5"
	"fmt"
	"strconv"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/steereddev/steered/internal/client"
	"github.com/steereddev/steered/internal/config"
	auracontext "github.com/steereddev/steered/internal/context"
	"github.com/steereddev/steered/internal/llm"
	"github.com/steereddev/steered/internal/model"
)

const probeInterval = 30
const maxMainViewIssues = 8

type status int

const (
	statusBooting status = iota
	statusProbing
	statusHealthy
	statusIssues
	statusError
	statusAnalyzing
)

// resourceAnalysisResult is sent when LLM analysis completes for a resource
type resourceAnalysisResult struct {
	key    string
	issues []*llm.Issue
	err    error
}

// Model is the bubbletea model for steered live TUI
type Model struct {
	client           *client.Client
	snapshot         *model.ClusterSnapshot
	issues           []Issue
	status           status
	probeCount       int
	lastProbe        time.Time
	nextProbe        int
	errors           []string
	probeTimeMs      int64
	analyzer         llm.Analyzer
	configManager    *config.Manager
	viewMode         string
	analyzing        map[string]bool
	resourceHashes   map[string]string
	copyConfirm      string
	copyConfirmIndex int
	isFirstProbe     bool
	termWidth        int
	termHeight       int
	viewport         viewport.Model
	vpReady          bool
	fixViewport      viewport.Model
	fixVpReady       bool
	llmStatus        string
}

// New creates a new TUI model
func New(c *client.Client, cfgManager *config.Manager) Model {
	m := Model{
		client:           c,
		status:           statusBooting,
		nextProbe:        probeInterval,
		analyzing:        make(map[string]bool),
		resourceHashes:   make(map[string]string),
		configManager:    cfgManager,
		viewMode:         "main",
		copyConfirmIndex: -1,
		isFirstProbe:     true,
		termWidth:        0,
		termHeight:       0,
		llmStatus:        "unconfigured",
	}

	cfg, err := cfgManager.LoadConfig()
	if err == nil && cfg.LLMProvider != "" {
		apiKey, _ := cfgManager.LoadAPIKey()
		m.analyzer = buildAnalyzer(cfg, apiKey)
		m.llmStatus = "connecting"
	}

	return m
}

// buildAnalyzer creates the right analyzer based on config
func buildAnalyzer(cfg *config.Config, apiKey string) llm.Analyzer {
	switch cfg.LLMProvider {
	case config.ProviderOllama:
		return llm.NewOllamaAnalyzer(cfg.LLMEndpoint, cfg.LLMModel)
	case config.ProviderOpenAI:
		return llm.NewOpenAIAnalyzer(cfg.LLMEndpoint, cfg.LLMModel, apiKey)
	case config.ProviderAnthropic:
		return llm.NewAnthropicAnalyzer(cfg.LLMEndpoint, cfg.LLMModel, apiKey)
	default:
		return nil
	}
}

// Init starts the first probe and ticker
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		probe(m.client),
		tick(),
	)
}

// updateViewportContent refreshes main viewport with latest issues
func (m *Model) updateViewportContent() {
	if m.vpReady {
		m.viewport.SetContent(renderIssuesSummary(*m))
	}
}

// updateFixViewportContent refreshes fix viewport with latest issues
func (m *Model) updateFixViewportContent() {
	if m.fixVpReady {
		m.fixViewport.SetContent(renderFixContent(*m))
	}
}

// recalculateViewport recalculates main viewport height
func (m *Model) recalculateViewport() {
	if m.termWidth == 0 || m.termHeight == 0 {
		return
	}

	top := "\n" +
		renderTUIHeader(*m) + "\n" +
		renderLiveBar(*m) + "\n" +
		renderHealthGrid(*m) + "\n\n"
	hint := renderPressAHint(*m)
	bottom := hint + renderTUIFooter(*m)

	fixedLines := lipgloss.Height(top) + lipgloss.Height(bottom)
	h := m.termHeight - fixedLines
	if h < 5 {
		h = 5
	}

	if !m.vpReady {
		m.viewport = viewport.New(m.termWidth, h)
		m.vpReady = true
	} else {
		m.viewport.Width = m.termWidth
		m.viewport.Height = h
	}
}

// recalculateFixViewport recalculates fix viewport height
func (m *Model) recalculateFixViewport() {
	if m.termWidth == 0 || m.termHeight == 0 {
		return
	}

	top := "\n" + renderFixHeader(*m) + "\n"
	bottom := renderFixFooter(*m)

	fixedLines := lipgloss.Height(top) + lipgloss.Height(bottom)
	h := m.termHeight - fixedLines
	if h < 5 {
		h = 5
	}

	if !m.fixVpReady {
		m.fixViewport = viewport.New(m.termWidth, h)
		m.fixVpReady = true
	} else {
		m.fixViewport.Width = m.termWidth
		m.fixViewport.Height = h
	}
}

// Update handles all messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var vpCmd tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		m.recalculateViewport()
		m.updateViewportContent()
		m.recalculateFixViewport()
		m.updateFixViewportContent()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.configManager != nil {
				m.configManager.Close()
			}
			return m, tea.Quit

		case "a":
			if m.viewMode == "main" {
				m.viewMode = "fix"
				m.recalculateFixViewport()
				m.updateFixViewportContent()
				return m, nil
			}

		case "j", "down":
			if m.viewMode == "main" && m.vpReady {
				m.viewport.LineDown(1)
				return m, nil
			}
			if m.viewMode == "fix" && m.fixVpReady {
				m.fixViewport.LineDown(1)
				return m, nil
			}

		case "k", "up":
			if m.viewMode == "main" && m.vpReady {
				m.viewport.LineUp(1)
				return m, nil
			}
			if m.viewMode == "fix" && m.fixVpReady {
				m.fixViewport.LineUp(1)
				return m, nil
			}

		case "r":
			if m.analyzer != nil && m.snapshot != nil {
				m.analyzing = make(map[string]bool)
				m.resourceHashes = make(map[string]string)
				m.viewport.GotoTop()
				m.fixViewport.GotoTop()
				m.updateViewportContent()
				m.updateFixViewportContent()
				return m, m.analyzeAllResources(m.snapshot)
			}

		case "esc", "b":
			m.viewMode = "main"
			m.copyConfirm = ""
			m.copyConfirmIndex = -1
			return m, nil

		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			if m.viewMode == "fix" {
				idx, _ := strconv.Atoi(msg.String())
				idx--
				if idx < len(m.issues) {
					issue := m.issues[idx]
					if issue.Command != "" {
						_ = clipboard.WriteAll(issue.Command)
						m.copyConfirm = "✓ copied to clipboard"
						m.copyConfirmIndex = idx
						m.updateFixViewportContent()
					}
				}
			}
		}

	case tickMsg:
		m.nextProbe--
		if m.nextProbe <= 0 {
			m.nextProbe = probeInterval
			m.status = statusProbing
			return m, tea.Batch(tick(), probe(m.client))
		}
		return m, tick()

	case probeDone:
		if msg.err != nil {
			m.status = statusError
			m.errors = append(m.errors, msg.err.Error())
			return m, nil
		}

		m.snapshot = msg.snapshot
		m.probeCount++
		m.lastProbe = time.Now()
		m.probeTimeMs = msg.durationMs

		m.issues = removeStaleIssues(m.issues, msg.snapshot)

		if severeIssues(m.issues) == 0 {
			m.status = statusHealthy
		} else {
			m.status = statusIssues
		}

		m.copyConfirm = ""

		m.copyConfirm = ""
		m.copyConfirmIndex = -1
		m.recalculateViewport()
		m.updateViewportContent()
		m.recalculateFixViewport()
		m.updateFixViewportContent()

		if m.analyzer != nil {
			if m.isFirstProbe {
				m.isFirstProbe = false
				return m, m.analyzeAllResources(msg.snapshot)
			} else {
				return m, m.analyzeChangedResources(msg.snapshot)
			}
		}

	case resourceAnalysisResult:
		delete(m.analyzing, msg.key)

		if msg.err != nil {
			m.llmStatus = "error"
		} else {
			m.llmStatus = "ok"
		}

		if msg.err == nil {
			m.issues = removeIssuesForResource(m.issues, msg.key)

			for _, li := range msg.issues {
				m.issues = append(m.issues, Issue{
					Severity:       li.Severity,
					ResourceType:   li.ResourceType,
					Title:          li.Title,
					Resource:       li.Resource,
					Namespace:      li.Namespace,
					Meta:           li.Meta,
					RootCause:      li.RootCause,
					FixExplanation: li.FixExplanation,
					Command:        li.Command,
					WatchFor:       li.WatchFor,
					Risk:           li.Risk,
					Confidence:     li.Confidence,
					Type:           li.Type,
					DetectedAt:     time.Now(),
				})
			}
		}

		if severeIssues(m.issues) == 0 && len(m.analyzing) == 0 {
			m.status = statusHealthy
		} else if severeIssues(m.issues) > 0 {
			m.status = statusIssues
		}

		m.recalculateViewport()

		m.recalculateViewport()
		m.updateViewportContent()
		m.recalculateFixViewport()
		m.updateFixViewportContent()
	}

	// pass remaining messages to active viewport
	if m.vpReady && m.viewMode == "main" {
		m.viewport, vpCmd = m.viewport.Update(msg)
	}
	if m.fixVpReady && m.viewMode == "fix" {
		m.fixViewport, vpCmd = m.fixViewport.Update(msg)
	}

	return m, vpCmd
}

// analyzeAllResources analyzes all resources in snapshot
func (m *Model) analyzeAllResources(snapshot *model.ClusterSnapshot) tea.Cmd {
	if m.analyzer == nil || snapshot == nil {
		return nil
	}

	resources := buildResourceList(snapshot)
	var cmds []tea.Cmd

	ctxBuilder := auracontext.New(m.client)
	ctx := context.Background()

	for _, r := range resources {
		if isSystemNamespace(r.Namespace) && r.Kind != "node" {
			continue
		}
		key := resourceKey(r.Kind, r.Name, r.Namespace)
		if m.analyzing[key] {
			continue
		}

		ic, err := ctxBuilder.Build(ctx, snapshot, r.Name, r.Namespace, r.Kind)
		if err == nil {
			m.resourceHashes[key] = resourceHash(ic.Events)
		}

		m.analyzing[key] = true
		m.status = statusAnalyzing
		cmds = append(cmds, m.analyzeResource(r.Kind, r.Name, r.Namespace, snapshot))
	}

	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

// analyzeChangedResources only analyzes resources whose state has changed
func (m *Model) analyzeChangedResources(snapshot *model.ClusterSnapshot) tea.Cmd {
	if m.analyzer == nil || snapshot == nil {
		return nil
	}

	resources := buildResourceList(snapshot)
	var cmds []tea.Cmd

	ctxBuilder := auracontext.New(m.client)
	ctx := context.Background()

	for _, r := range resources {
		if isSystemNamespace(r.Namespace) && r.Kind != "node" {
			continue
		}

		key := resourceKey(r.Kind, r.Name, r.Namespace)
		if m.analyzing[key] {
			continue
		}

		ic, err := ctxBuilder.Build(ctx, snapshot, r.Name, r.Namespace, r.Kind)
		if err != nil {
			continue
		}

		currentHash := resourceHash(ic.Events)
		previousHash := m.resourceHashes[key]

		if currentHash != previousHash {
			m.analyzing[key] = true
			m.resourceHashes[key] = currentHash
			cmds = append(cmds, m.analyzeResource(r.Kind, r.Name, r.Namespace, snapshot))
		}
	}

	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

// analyzeResource runs LLM analysis for a single resource
func (m *Model) analyzeResource(kind, name, namespace string, snapshot *model.ClusterSnapshot) tea.Cmd {
	analyzer := m.analyzer
	c := m.client
	key := resourceKey(kind, name, namespace)

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		ctxBuilder := auracontext.New(c)
		ic, err := ctxBuilder.Build(ctx, snapshot, name, namespace, kind)
		if err != nil {
			return resourceAnalysisResult{key: key, err: err}
		}

		issues, err := analyzer.Analyze(ctx, ic)
		return resourceAnalysisResult{
			key:    key,
			issues: issues,
			err:    err,
		}
	}
}

// removeIssuesForResource removes all issues for a resource key
func removeIssuesForResource(issues []Issue, key string) []Issue {
	parts := splitResourceKey(key)
	if len(parts) != 3 {
		return issues
	}
	kind, name, namespace := parts[0], parts[1], parts[2]

	var filtered []Issue
	for _, i := range issues {
		if !(i.ResourceType == kind && i.Resource == name && i.Namespace == namespace) {
			filtered = append(filtered, i)
		}
	}
	return filtered
}

// splitResourceKey splits a resource key into kind/name/namespace
func splitResourceKey(key string) []string {
	parts := make([]string, 0)
	start := 0
	count := 0
	for i, c := range key {
		if c == '/' {
			parts = append(parts, key[start:i])
			start = i + 1
			count++
			if count == 2 {
				parts = append(parts, key[start:])
				return parts
			}
		}
	}
	return parts
}

// computeHash computes md5 hash of string slice
func computeHash(data []string) string {
	h := md5.New()
	for _, d := range data {
		h.Write([]byte(d))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// View renders the current state
func (m Model) View() string {
	return render(m)
}
