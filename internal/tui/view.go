package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/steereddev/steered/internal/model"
)

const steeredVersion = "v2.1.4"

var (
	tColorGreen  = lipgloss.Color("#3fb950")
	tColorAmber  = lipgloss.Color("#d29922")
	tColorRed    = lipgloss.Color("#f85149")
	tColorBlue   = lipgloss.Color("#58a6ff")
	tColorPurple = lipgloss.Color("#bc8cff")
	tColorMuted  = lipgloss.Color("#484f58")
	tColorBright = lipgloss.Color("#e6edf3")
	tColorBgGrid = lipgloss.Color("#21262d")
	tColorBgSub  = lipgloss.Color("#161b22")
)

var (
	tStyleOk     = lipgloss.NewStyle().Foreground(tColorGreen)
	tStyleWarn   = lipgloss.NewStyle().Foreground(tColorAmber)
	tStyleErr    = lipgloss.NewStyle().Foreground(tColorRed)
	tStyleBlue   = lipgloss.NewStyle().Foreground(tColorBlue)
	tStylePurple = lipgloss.NewStyle().Foreground(tColorPurple)
	tStyleMuted  = lipgloss.NewStyle().Foreground(tColorMuted)
	tStyleBright = lipgloss.NewStyle().Foreground(tColorBright)

	tStyleDivider = lipgloss.NewStyle().
			Foreground(tColorBgGrid)
)

func tWidth(w int) int {
	if w > 10 {
		return w
	}
	return 120
}

func tDivider(w int) string {
	return tStyleDivider.Render(strings.Repeat("─", tWidth(w)))
}

func boxStyle(borderColor string, w int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Padding(0, 1).
		Width(tWidth(w) - 4)
}

func render(m Model) string {
	if m.viewMode == "fix" {
		return renderFixView(m)
	}
	if m.analyzer == nil || m.llmStatus == "error" {
		return renderLLMGateView(m)
	}
	if m.llmStatus == "connecting" {
		return renderConnectingView(m)
	}
	return renderMainView(m)
}

func renderLLMGateView(m Model) string {
	w := m.termWidth
	h := m.termHeight

	var title, message string
	var borderColor string

	if m.analyzer == nil {
		borderColor = "#484f58"
		title = tStyleWarn.Render("⚡  steered requires an LLM to analyze your cluster")
		message = tStyleMuted.Render("no LLM configured")
	} else {
		borderColor = "#f8514933"
		name := m.analyzer.Name()
		if idx := strings.Index(name, "/"); idx != -1 {
			name = name[idx+1:]
		}
		title = tStyleErr.Render("✗  LLM connection failed")
		message = tStyleMuted.Render(name)
	}

	// blink toggle using tick
	blink := m.nextProbe%2 == 0

	var content strings.Builder
	content.WriteString("\n")
	content.WriteString("    " + title + "\n")
	content.WriteString("\n")
	if blink {
		content.WriteString("    " + message + "\n")
		content.WriteString("    " + tStyleMuted.Render("check your API key and model name") + "\n")
	} else {
		content.WriteString("\n")
		content.WriteString("\n")
	}
	content.WriteString("\n")
	if m.analyzer == nil {
		content.WriteString("    " + tStyleMuted.Render("run  ") + tStyleBlue.Render("steered --setup") + tStyleMuted.Render("  to get started") + "\n")
	} else {
		content.WriteString("    " + tStyleMuted.Render("run  ") + tStyleBlue.Render("steered --setup") + tStyleMuted.Render("  to reconfigure") + "\n")
		content.WriteString("    " + tStyleMuted.Render("run  ") + tStyleBlue.Render("steered --clear") + tStyleMuted.Render("  to reset") + "\n")
	}
	content.WriteString("\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Padding(1, 2).
		Width(tWidth(w) - 4).
		Render(content.String())

	// vertically center the box
	header := renderTUIHeader(m)
	footer := renderTUIFooter(m)

	headerLines := lipgloss.Height(header)
	footerLines := lipgloss.Height(footer)
	boxLines := lipgloss.Height(box)

	availableLines := h - headerLines - footerLines - boxLines - 2
	topPadding := availableLines / 2
	if topPadding < 1 {
		topPadding = 1
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("\n", topPadding))
	b.WriteString(box)
	b.WriteString(strings.Repeat("\n", topPadding))
	b.WriteString(footer)

	return b.String()
}

func renderConnectingView(m Model) string {
	w := m.termWidth
	h := m.termHeight

	name := ""
	if m.analyzer != nil {
		name = m.analyzer.Name()
		if idx := strings.Index(name, "/"); idx != -1 {
			name = name[idx+1:]
		}
	}

	// blink toggle
	blink := m.nextProbe%2 == 0

	var content strings.Builder
	content.WriteString("\n")
	if blink {
		content.WriteString("    " + tStyleWarn.Render("⟳  connecting to "+name+" ...") + "\n")
	} else {
		content.WriteString("\n")
	}
	content.WriteString("\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3fb95033")).
		Padding(1, 2).
		Width(tWidth(w) - 4).
		Render(content.String())

	header := renderTUIHeader(m)
	footer := renderTUIFooter(m)

	headerLines := lipgloss.Height(header)
	footerLines := lipgloss.Height(footer)
	boxLines := lipgloss.Height(box)

	availableLines := h - headerLines - footerLines - boxLines - 2
	topPadding := availableLines / 2
	if topPadding < 1 {
		topPadding = 1
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("\n", topPadding))
	b.WriteString(box)
	b.WriteString(strings.Repeat("\n", topPadding))
	b.WriteString(footer)

	return b.String()
}

// ── MAIN VIEW ────────────────────────────────────────────────────────────────

// ── MAIN VIEW ────────────────────────────────────────────────────────────────

func renderMainView(m Model) string {
	fixedTop := "\n" +
		renderTUIHeader(m) + "\n" +
		renderLiveBar(m) + "\n" +
		renderHealthGrid(m) + "\n\n"

	hint := renderPressAHint(m)
	fixedBottom := hint + renderTUIFooter(m)

	if !m.vpReady {
		return fixedTop + fixedBottom
	}

	return fixedTop + m.viewport.View() + "\n" + fixedBottom
}

func renderPressAHint(m Model) string {
	if len(m.issues) == 0 {
		return ""
	}
	w := m.termWidth
	hint := tStyleOk.Render("⚡  press ") +
		tStyleBlue.Render("'a'") +
		tStyleOk.Render(" for detailed fix guidance")
	return boxStyle("#3fb95033", w).Render(hint) + "\n"
}

func renderIssuesSummary(m Model) string {
	w := m.termWidth
	var b strings.Builder

	sectionTitle := func(title string) string {
		hint := tStyleBlue.Render("↑↓") + tStyleMuted.Render(" to scroll")
		hintWidth := lipgloss.Width("↑↓ to scroll")
		titleWidth := lipgloss.Width(title)
		padLen := tWidth(w) - 4 - titleWidth - hintWidth - 3
		if padLen < 1 {
			padLen = 1
		}
		return title + strings.Repeat(" ", padLen) + hint
	}

	if m.analyzer == nil {
		content := tStyleMuted.Render("⚡  configure LLM to enable AI analysis  ·  run ") +
			tStyleBlue.Render("steered --setup")
		b.WriteString(boxStyle("#21262d", w).Render(content) + "\n")
		return b.String()
	}

	if m.status == statusAnalyzing || (len(m.analyzing) > 0 && len(m.issues) == 0) {
		analyzing := len(m.analyzing)
		content := tStyleWarn.Render("⟳  analyzing cluster") +
			tStyleMuted.Render(fmt.Sprintf("  ·  %d resources", analyzing))
		b.WriteString(boxStyle("#d2992233", w).Render(content) + "\n\n")
	}

	if len(m.issues) == 0 && len(m.analyzing) == 0 {
		b.WriteString(boxStyle("#3fb95066", w).Render(tStyleOk.Render("✓  cluster is healthy — no issues found")) + "\n\n")
		return b.String()
	}

	var critical, security, warning []Issue
	for _, i := range m.issues {
		switch i.Severity {
		case "critical":
			critical = append(critical, i)
		case "security":
			security = append(security, i)
		default:
			warning = append(warning, i)
		}
	}

	issueNum := 1

	if len(critical) > 0 {
		b.WriteString(sectionTitle(tStyleErr.Render(fmt.Sprintf("MUST FIX (%d)", len(critical)))) + "\n")
		var inner strings.Builder
		for i, issue := range critical {
			inner.WriteString(renderIssueSummaryRow(issue, issueNum))
			issueNum++
			if i < len(critical)-1 {
				inner.WriteString(tStyleDivider.Render(strings.Repeat("─", tWidth(w)-12)) + "\n")
			}
		}
		b.WriteString(boxStyle("#f8514933", w).Render(inner.String()) + "\n\n")
	}

	if len(security) > 0 {
		b.WriteString(sectionTitle(tStylePurple.Render(fmt.Sprintf("SECURITY (%d)", len(security)))) + "\n")
		var inner strings.Builder
		for i, issue := range security {
			inner.WriteString(renderIssueSummaryRow(issue, issueNum))
			issueNum++
			if i < len(security)-1 {
				inner.WriteString(tStyleDivider.Render(strings.Repeat("─", tWidth(w)-12)) + "\n")
			}
		}
		b.WriteString(boxStyle("#bc8cff33", w).Render(inner.String()) + "\n\n")
	}

	if len(warning) > 0 {
		b.WriteString(sectionTitle(tStyleWarn.Render(fmt.Sprintf("GOOD PRACTICE (%d)", len(warning)))) + "\n")
		var inner strings.Builder
		for i, issue := range warning {
			inner.WriteString(renderIssueSummaryRow(issue, issueNum))
			issueNum++
			if i < len(warning)-1 {
				inner.WriteString(tStyleDivider.Render(strings.Repeat("─", tWidth(w)-12)) + "\n")
			}
		}
		b.WriteString(boxStyle("#d2992233", w).Render(inner.String()) + "\n\n")
	}

	return b.String()
}

func renderIssueSummaryRow(issue Issue, num int) string {
	var icon string
	var titleStyle lipgloss.Style

	switch issue.Severity {
	case "critical":
		icon = tStyleErr.Render("!")
		titleStyle = tStyleErr
	case "security":
		icon = tStylePurple.Render("⚠")
		titleStyle = tStylePurple
	default:
		icon = tStyleWarn.Render("▲")
		titleStyle = tStyleWarn
	}

	numStr := tStyleMuted.Render(fmt.Sprintf("%d", num))
	resType := tStyleMuted.Render(issue.ResourceType + ":")
	title := titleStyle.Render(issue.Title)
	meta := tStyleMuted.Render(issue.Meta)

	return fmt.Sprintf("%s  %s  %s  %s\n   %s\n   %s\n",
		icon, numStr, resType, title,
		resourceLocation(issue),
		meta)
}

// ── FIX VIEW ─────────────────────────────────────────────────────────────────

func renderFixView(m Model) string {
	fixedTop := "\n" + renderFixHeader(m) + "\n"
	fixedBottom := renderFixFooter(m)

	if !m.fixVpReady {
		return fixedTop + fixedBottom
	}

	return fixedTop + m.fixViewport.View() + "\n" + fixedBottom
}

func renderFixHeader(m Model) string {
	w := m.termWidth

	brand := tStyleBright.Render("▸  S T E E R E D") +
		"  " + tStyleOk.Render("FIX GUIDANCE")
	tagline := lipgloss.NewStyle().Foreground(tColorMuted).Italic(true).Render("the light that guides you through darkness")

	providerName := ""
	if m.analyzer != nil {
		providerName = tStyleMuted.Render(m.analyzer.Name())
	}

	clusterName := tStyleBlue.Render(m.client.ClusterName)
	ctx := tStyleMuted.Render(m.client.Context)
	ts := tStyleMuted.Render(time.Now().UTC().Format("2006-01-02  15:04:05 UTC"))

	left := brand + "\n" + tagline
	right := fmt.Sprintf("cluster: %s\ncontext: %s\n%s  %s",
		clusterName, ctx, ts, providerName)

	rw := 55
	lw := tWidth(w) - 8 - rw

	lb := lipgloss.NewStyle().Width(lw).Render(left)
	rb := lipgloss.NewStyle().Width(rw).Align(lipgloss.Right).Render(right)

	return boxStyle("#3fb95066", w).Render(lipgloss.JoinHorizontal(lipgloss.Top, lb, rb))
}

func renderFixContent(m Model) string {
	w := m.termWidth
	var inner strings.Builder

	if len(m.issues) == 0 {
		if len(m.analyzing) > 0 {
			inner.WriteString(tStyleWarn.Render("⟳  analyzing cluster issues...\n"))
		} else {
			inner.WriteString(tStyleOk.Render("✓  cluster is healthy — no issues found\n"))
		}
		return boxStyle("#3fb95066", w).Render(inner.String())
	}

	for i, issue := range m.issues {
		var icon string
		var titleStyle lipgloss.Style

		switch issue.Severity {
		case "critical":
			icon = tStyleErr.Render("!")
			titleStyle = tStyleErr
		case "security":
			icon = tStylePurple.Render("⚠")
			titleStyle = tStylePurple
		default:
			icon = tStyleWarn.Render("▲")
			titleStyle = tStyleWarn
		}

		num := tStyleMuted.Render(fmt.Sprintf("%d", i+1))
		resType := tStyleMuted.Render(issue.ResourceType + ":")

		inner.WriteString(fmt.Sprintf("%s  %s  %s  %s\n",
			icon, num, resType, titleStyle.Render(issue.Title)))
		inner.WriteString(fmt.Sprintf("   %s\n", resourceLocation(issue)))

		if issue.RootCause != "" {
			inner.WriteString(fmt.Sprintf("   %s  %s\n",
				tStyleMuted.Render("WHY:     "),
				tStyleBright.Render(truncate(issue.RootCause, 90)),
			))
		}

		if issue.FixExplanation != "" {
			label := "ACTION:  "
			if issue.Type == "investigate" {
				label = "LOOK FOR:"
			}
			inner.WriteString(fmt.Sprintf("   %s  %s\n",
				tStyleMuted.Render(label),
				tStyleBright.Render(truncate(issue.FixExplanation, 90)),
			))
		}

		if issue.Command != "" {
			displayCmd := issue.Command
			if idx := strings.Index(displayCmd, "\n"); idx != -1 {
				displayCmd = displayCmd[:idx]
			}
			if len(displayCmd) > 88 {
				displayCmd = displayCmd[:85] + "..."
			}

			if issue.Type == "investigate" {
				inner.WriteString(fmt.Sprintf("   %s  %s\n",
					tStyleBlue.Render("🔍 CHECK:"),
					tStyleBlue.Render(displayCmd),
				))
			} else {
				inner.WriteString(fmt.Sprintf("   %s  %s\n",
					tStyleOk.Render("✅ FIX:  "),
					tStyleBlue.Render(displayCmd),
				))
			}

			copyWidth := tWidth(w) - 20
			var copyHint string
			if m.copyConfirmIndex == i && m.copyConfirm != "" {
				confirmText := "✓ copied to clipboard"
				padding := copyWidth - len(confirmText) - 4
				if padding < 0 {
					padding = 0
				}
				copyHint = tStyleOk.Render(
					"   ╰─ " + confirmText + strings.Repeat("─", padding) + "╯",
				)
			} else {
				hintText := fmt.Sprintf("press '%d' to copy command", i+1)
				padding := copyWidth - len(hintText) - 4
				if padding < 0 {
					padding = 0
				}
				if issue.Type == "investigate" {
					copyHint = tStyleBlue.Render(
						"   ╰─ " + hintText + strings.Repeat("─", padding) + "╯",
					)
				} else {
					copyHint = tStyleWarn.Render(
						"   ╰─ " + hintText + strings.Repeat("─", padding) + "╯",
					)
				}
			}
			inner.WriteString(copyHint + "\n")
		}

		if issue.Risk != "" {
			inner.WriteString(fmt.Sprintf("   %s  %s\n",
				tStyleMuted.Render("RISK:    "),
				tStyleWarn.Render(truncate(issue.Risk, 90)),
			))
		}

		if issue.Confidence != "" {
			inner.WriteString(fmt.Sprintf("   %s  %s\n",
				tStyleMuted.Render("CONFIDENCE:"),
				confidenceStyle(issue.Confidence).Render(issue.Confidence),
			))
		}

		if i < len(m.issues)-1 {
			inner.WriteString(tStyleDivider.Render(strings.Repeat("─", tWidth(w)-10)) + "\n")
		}
	}

	return boxStyle("#3fb95066", w).Render(inner.String())
}

func renderFixFooter(m Model) string {
	w := m.termWidth

	left := tStyleMuted.Render("press ") +
		tStyleBlue.Render("'esc'") +
		tStyleMuted.Render(" to return  ·  ") +
		tStyleBlue.Render("'r'") +
		tStyleMuted.Render(" to re-analyze  ·  ") +
		tStyleWarn.Render("1-9") +
		tStyleMuted.Render(" to copy fix  ·  ") +
		tStyleBlue.Render("↑↓") +
		tStyleMuted.Render(" to scroll")

	scrollInfo := ""
	if m.fixVpReady && m.fixViewport.ScrollPercent() > 0 {
		scrollInfo = tStyleMuted.Render(fmt.Sprintf("  ·  %.0f%%", m.fixViewport.ScrollPercent()*100))
	}

	right := tStyleMuted.Render(fmt.Sprintf("probe #%d  ·  ", m.probeCount)) +
		tStyleOk.Render(fmt.Sprintf("%d issues", len(m.issues))) +
		scrollInfo

	frw := 30
	flw := tWidth(w) - 8 - frw

	flb := lipgloss.NewStyle().Width(flw).Render(left)
	frb := lipgloss.NewStyle().Width(frw).Align(lipgloss.Right).Render(right)

	content := lipgloss.JoinHorizontal(lipgloss.Top, flb, frb)
	return boxStyle("#3fb95033", w).Render(content)
}

// ── SHARED ───────────────────────────────────────────────────────────────────

func renderTUIHeader(m Model) string {
	w := m.termWidth
	brand := tStyleBright.Render("▸  S T E E R E D")
	tagline := lipgloss.NewStyle().Foreground(tColorMuted).Italic(true).Render("the light that guides you through darkness")

	clusterName := tStyleBlue.Render(m.client.ClusterName)
	ctx := tStyleMuted.Render(m.client.Context)
	ts := tStyleMuted.Render(time.Now().UTC().Format("2006-01-02  15:04:05 UTC"))

	llmLine := ""
	if m.analyzer != nil {
		name := m.analyzer.Name()
		if idx := strings.Index(name, "/"); idx != -1 {
			name = name[idx+1:]
		}
		switch m.llmStatus {
		case "ok":
			llmLine = name + "  " + tStyleOk.Render("✓")
		case "error":
			llmLine = name + "  " + tStyleErr.Render("✗")
		default:
			llmLine = name + "  " + tStyleMuted.Render("…")
		}
	}

	left := brand + "\n" + tagline
	right := fmt.Sprintf("cluster: %s\ncontext: %s\n%s\n%s", clusterName, ctx, ts, llmLine)

	rw := 50
	lw := tWidth(w) - 8 - rw

	lb := lipgloss.NewStyle().Width(lw).Render(left)
	rb := lipgloss.NewStyle().Width(rw).Align(lipgloss.Right).Render(right)

	content := lipgloss.JoinHorizontal(lipgloss.Top, lb, rb)
	return boxStyle("#e6edf3", w).Render(content)
}

func renderLiveBar(m Model) string {
	w := m.termWidth
	var dot string
	if m.nextProbe%2 == 0 {
		dot = tStyleOk.Render("●")
	} else {
		dot = tStyleMuted.Render("●")
	}

	liveLabel := tStyleOk.Render("LIVE")
	interval := tStyleMuted.Render("probing every ") + tStyleBlue.Render("30s")

	healthPct := 100
	if m.snapshot != nil {
		total := len(m.snapshot.Nodes) + len(m.snapshot.Deployments) + len(m.snapshot.Pods)
		if total > 0 {
			severeCount := 0
			for _, i := range m.issues {
				if i.Severity == "critical" || i.Severity == "security" {
					severeCount++
				}
			}
			healthPct = 100 - (severeCount * 100 / total)
			if healthPct < 0 {
				healthPct = 0
			}
		}
	}

	var healthState string
	switch m.status {
	case statusBooting:
		healthState = tStyleMuted.Render("  ·  initializing...")
	case statusProbing:
		healthState = tStyleWarn.Render("  ·  probing cluster...")
	case statusAnalyzing:
		healthState = tStyleWarn.Render("  ·  ⟳ analyzing with AI...")
	case statusHealthy:
		healthState = tStyleOk.Render(fmt.Sprintf("  ·  ✓ cluster healthy  %d%%", healthPct))
	case statusIssues:
		pctStyle := tStyleWarn
		if healthPct < 50 {
			pctStyle = tStyleErr
		}
		severe := severeIssues(m.issues)
		healthState = tStyleErr.Render(fmt.Sprintf("  ·  %d issues found  ", severe)) +
			pctStyle.Render(fmt.Sprintf("%d%% healthy", healthPct))
	case statusError:
		healthState = tStyleErr.Render("  ·  connection error")
	}

	left := dot + "  " + liveLabel + "  " + interval + healthState

	nextProbe := tStyleMuted.Render("next probe in ") + tStyleBlue.Render(fmt.Sprintf("%ds", m.nextProbe))
	lastProbe := ""
	if !m.lastProbe.IsZero() {
		lastProbe = tStyleMuted.Render("  ·  last probe ") + tStyleBlue.Render(m.lastProbe.Format("15:04:05"))
	}

	right := nextProbe + lastProbe

	rw := 45
	lw := tWidth(w) - 8 - rw

	lb := lipgloss.NewStyle().Width(lw).Render(left)
	rb := lipgloss.NewStyle().Width(rw).Align(lipgloss.Right).Render(right)

	content := lipgloss.JoinHorizontal(lipgloss.Top, lb, rb)
	return boxStyle("#3fb95033", w).Render(content)
}

func renderHealthGrid(m Model) string {
	w := m.termWidth
	if m.snapshot == nil {
		return boxStyle("#3fb95033", w).Render(tStyleMuted.Render("  waiting for first probe..."))
	}

	s := m.snapshot

	running, pending, failed := 0, 0, 0
	for _, p := range s.Pods {
		switch strings.ToLower(p.Status) {
		case "running":
			running++
		case "pending":
			pending++
		case "failed", "error", "crashloopbackoff":
			failed++
		}
	}

	readyNodes, warnNodes := 0, 0
	for _, n := range s.Nodes {
		if strings.ToLower(n.Status) == "ready" {
			readyNodes++
		} else {
			warnNodes++
		}
	}

	healthyDeploys, downDeploys := 0, 0
	for _, d := range s.Deployments {
		if d.Available > 0 {
			healthyDeploys++
		} else {
			downDeploys++
		}
	}

	type cell struct {
		num  string
		lbl  string
		sub  string
		nclr lipgloss.Style
	}

	cells := []cell{
		{fmt.Sprintf("%d", len(s.Nodes)), "nodes", fmt.Sprintf("%d ready · %d warn", readyNodes, warnNodes), tStyleBlue},
		{fmt.Sprintf("%d", len(s.Namespaces)), "namespaces", "active", tStyleOk},
		{fmt.Sprintf("%d", running), "pods running", fmt.Sprintf("%d pending · %d failed", pending, failed), tStyleOk},
		{fmt.Sprintf("%d", len(s.Deployments)), "deployments", fmt.Sprintf("%d healthy · %d down", healthyDeploys, downDeploys), tStylePurple},
		{fmt.Sprintf("%d", len(s.Services)), "services", fmt.Sprintf("%d external", externalSvcs(s)), tStyleBlue},
		{fmt.Sprintf("%d", len(s.Ingresses)), "ingresses", "all routes", tStyleBlue},
		{fmt.Sprintf("%d", len(s.PVCs)), "pvcs", "persistent volumes", tStyleWarn},
		{fmt.Sprintf("%d", severeIssues(m.issues)), "issues", fmt.Sprintf("%d analyzing", len(m.analyzing)), tStyleWarn},
	}

	colW := (tWidth(w) - 8) / 4

	cellStyle := func(last bool) lipgloss.Style {
		st := lipgloss.NewStyle().
			Width(colW).
			PaddingLeft(2).
			PaddingRight(1)
		if !last {
			st = st.BorderRight(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(tColorBgGrid)
		}
		return st
	}

	buildRow := func(rowCells []cell) string {
		var blocks []string
		for i, c := range rowCells {
			line1 := c.nclr.Render(c.num) + "  " + tStyleMuted.Render(strings.ToUpper(c.lbl))
			line2 := tStyleMuted.Render(c.sub)
			content := line1 + "\n" + line2
			blocks = append(blocks, cellStyle(i == len(rowCells)-1).Render(content))
		}
		return lipgloss.JoinHorizontal(lipgloss.Top, blocks...)
	}

	divider := tStyleDivider.Render(strings.Repeat("─", tWidth(w)-6))

	var health strings.Builder
	health.WriteString(buildRow(cells[:4]) + "\n")
	health.WriteString(divider + "\n")
	health.WriteString(buildRow(cells[4:]))

	return boxStyle("#3fb95033", w).Render(health.String())
}

func renderTUIFooter(m Model) string {
	w := m.termWidth

	left := tStyleMuted.Render("steered "+steeredVersion+"  ·  steered.dev  ·  ") +
		tStyleBlue.Render("ctrl+c") +
		tStyleMuted.Render(" to exit  ·  ") +
		tStyleBlue.Render("↑↓") +
		tStyleMuted.Render(" to scroll")

	scrollInfo := ""
	if m.vpReady && m.viewport.ScrollPercent() > 0 {
		scrollInfo = tStyleMuted.Render(fmt.Sprintf("  ·  %.0f%%", m.viewport.ScrollPercent()*100))
	}

	right := tStyleMuted.Render(fmt.Sprintf("probe #%d  ·  collected in ", m.probeCount)) +
		tStyleOk.Render(fmt.Sprintf("%dms", m.probeTimeMs)) +
		tStyleMuted.Render(fmt.Sprintf("  ·  %d analyzing", len(m.analyzing))) +
		scrollInfo

	rw := 55
	lw := tWidth(w) - 8 - rw

	lb := lipgloss.NewStyle().Width(lw).Render(left)
	rb := lipgloss.NewStyle().Width(rw).Align(lipgloss.Right).Render(right)

	content := lipgloss.JoinHorizontal(lipgloss.Top, lb, rb)
	return boxStyle("#3fb95033", w).Render(content)
}

func externalSvcs(s *model.ClusterSnapshot) int {
	count := 0
	for _, svc := range s.Services {
		if svc.ExternalIP != "" && svc.ExternalIP != "<none>" {
			count++
		}
	}
	return count
}

func resourceLocation(issue Issue) string {
	if issue.ResourceType == "namespace" {
		return tStyleMuted.Render("namespace: ") + tStyleBright.Render(issue.Resource)
	}
	return tStyleMuted.Render("name: ") + tStyleBright.Render(issue.Resource) +
		tStyleMuted.Render("   namespace: ") + tStyleBright.Render(issue.Namespace)
}

func confidenceStyle(confidence string) lipgloss.Style {
	switch confidence {
	case "high":
		return tStyleOk
	case "medium":
		return tStyleWarn
	default:
		return tStyleErr
	}
}
