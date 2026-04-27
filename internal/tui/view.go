package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/steereddev/steered/internal/model"
)

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

	tStyleBoxCrit = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#f8514933")).
			Padding(0, 1).
			Width(tWidth() - 4)

	tStyleBoxWarn = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#d2992233")).
			Padding(0, 1).
			Width(tWidth() - 4)

	tStyleBoxOk = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3fb95066")).
			Padding(0, 1).
			Width(tWidth() - 4)

	tStyleBoxHeader = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#e6edf3")).
			Padding(0, 1).
			Width(tWidth() - 4)

	tStyleBoxLive = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3fb95033")).
			Padding(0, 1).
			Width(tWidth() - 4)

	tStyleBoxHealth = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(tColorBgGrid).
			Padding(0, 1).
			Width(tWidth() - 4)

	tStyleDivider = lipgloss.NewStyle().
			Foreground(tColorBgGrid)
)

func tWidth() int {
	return 120
}

func tDivider() string {
	return tStyleDivider.Render(strings.Repeat("─", tWidth()))
}

func render(m Model) string {
	if m.viewMode == "fix" {
		return renderFixView(m)
	}
	return renderMainView(m)
}

func renderMainView(m Model) string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(renderTUIHeader(m))
	b.WriteString("\n")
	b.WriteString(renderLiveBar(m))
	b.WriteString("\n")
	b.WriteString(renderHealthGrid(m))
	b.WriteString("\n")
	b.WriteString(tDivider())
	b.WriteString("\n")
	b.WriteString(renderIssuesSummary(m))
	b.WriteString(renderResolved(m))
	b.WriteString(tDivider())
	b.WriteString("\n")
	b.WriteString(renderTUIFooter(m))

	return b.String()
}

func renderIssuesSummary(m Model) string {
	var b strings.Builder

	if m.analyzer == nil {
		content := tStyleMuted.Render("⚡  configure LLM to enable AI analysis  ·  run ") +
			tStyleBlue.Render("steered --setup")
		b.WriteString(lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(tColorBgGrid).
			Padding(0, 1).
			Width(tWidth()-4).
			Render(content) + "\n")
		return b.String()
	}

	if m.status == statusAnalyzing || (len(m.analyzing) > 0 && len(m.issues) == 0) {
		analyzing := len(m.analyzing)
		content := tStyleWarn.Render("⟳  analyzing cluster") +
			tStyleMuted.Render(fmt.Sprintf("  ·  %d resources", analyzing))
		b.WriteString(lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#d2992233")).
			Padding(0, 1).
			Width(tWidth()-4).
			Render(content) + "\n\n")
	}

	if len(m.issues) == 0 && len(m.analyzing) == 0 {
		b.WriteString(tStyleBoxOk.Render(tStyleOk.Render("✓  cluster is healthy — no issues found")) + "\n\n")
		return b.String()
	}

	// show top N issues
	displayed := topIssues(m.issues, maxMainViewIssues)
	total := len(m.issues)

	// section headers
	var critical, security, warning []Issue
	for _, i := range displayed {
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

	// critical
	if len(critical) > 0 {
		mustFixTitle := tStyleErr.Render("MUST FIX") +
			"  " + tStyleDivider.Render(strings.Repeat("─", tWidth()-30)) +
			"  " + tStyleMuted.Render(fmt.Sprintf("%d critical", len(critical)))
		b.WriteString(mustFixTitle + "\n\n")

		var inner strings.Builder
		for i, issue := range critical {
			inner.WriteString(renderIssueSummaryRow(issue, issueNum))
			issueNum++
			if i < len(critical)-1 {
				inner.WriteString(tStyleDivider.Render(strings.Repeat("─", tWidth()-12)) + "\n")
			}
		}
		b.WriteString(tStyleBoxCrit.Render(inner.String()) + "\n\n")
	}

	// warnings
	if len(warning) > 0 {
		goodTitle := tStyleWarn.Render("GOOD PRACTICE") +
			"  " + tStyleDivider.Render(strings.Repeat("─", tWidth()-36)) +
			"  " + tStyleMuted.Render(fmt.Sprintf("%d recommendations", len(warning)))
		b.WriteString(goodTitle + "\n\n")

		var inner strings.Builder
		for i, issue := range warning {
			inner.WriteString(renderIssueSummaryRow(issue, issueNum))
			issueNum++
			if i < len(warning)-1 {
				inner.WriteString(tStyleDivider.Render(strings.Repeat("─", tWidth()-12)) + "\n")
			}
		}
		b.WriteString(tStyleBoxWarn.Render(inner.String()) + "\n\n")
	}

	// security
	if len(security) > 0 {
		tStyleBoxSec := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#bc8cff33")).
			Padding(0, 1).
			Width(tWidth() - 4)

		secTitle := tStylePurple.Render("SECURITY") +
			"  " + tStyleDivider.Render(strings.Repeat("─", tWidth()-28)) +
			"  " + tStyleMuted.Render(fmt.Sprintf("%d findings", len(security)))
		b.WriteString(secTitle + "\n\n")

		var inner strings.Builder
		for i, issue := range security {
			inner.WriteString(renderIssueSummaryRow(issue, issueNum))
			issueNum++
			if i < len(security)-1 {
				inner.WriteString(tStyleDivider.Render(strings.Repeat("─", tWidth()-12)) + "\n")
			}
		}
		b.WriteString(tStyleBoxSec.Render(inner.String()) + "\n\n")
	}

	// show more hint
	if total > maxMainViewIssues {
		more := total - maxMainViewIssues
		hint := tStyleMuted.Render(fmt.Sprintf("  %d more issues — press ", more)) +
			tStyleBlue.Render("'a'") +
			tStyleMuted.Render(" to see all")
		b.WriteString(lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(tColorBgGrid).
			Padding(0, 1).
			Width(tWidth()-4).
			Render(hint) + "\n\n")
	} else if len(m.issues) > 0 {
		hint := tStyleOk.Render("⚡  press ") +
			tStyleBlue.Render("'a'") +
			tStyleOk.Render(" for detailed fix guidance")
		b.WriteString(lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3fb95033")).
			Padding(0, 1).
			Width(tWidth()-4).
			Render(hint) + "\n")
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

func renderFixView(m Model) string {
	var b strings.Builder

	b.WriteString("\n")

	// header
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

	w := tWidth() - 8
	rw := 55
	lw := w - rw

	lb := lipgloss.NewStyle().Width(lw).Render(left)
	rb := lipgloss.NewStyle().Width(rw).Align(lipgloss.Right).Render(right)

	b.WriteString(lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3fb95066")).
		Padding(0, 1).
		Width(tWidth() - 4).
		Render(lipgloss.JoinHorizontal(lipgloss.Top, lb, rb)))
	b.WriteString("\n")

	// fix content
	tStyleBoxLLM := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3fb95066")).
		Padding(0, 1).
		Width(tWidth() - 4)

	var inner strings.Builder

	if len(m.issues) == 0 {
		if len(m.analyzing) > 0 {
			inner.WriteString(tStyleWarn.Render("⟳  analyzing cluster issues...\n"))
		} else {
			inner.WriteString(tStyleOk.Render("✓  cluster is healthy — no issues found\n"))
		}
	} else {
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

			// title line
			inner.WriteString(fmt.Sprintf("%s  %s  %s  %s\n",
				icon, num, resType, titleStyle.Render(issue.Title)))

			// resource location
			inner.WriteString(fmt.Sprintf("   %s\n", resourceLocation(issue)))

			// WHY
			if issue.RootCause != "" {
				inner.WriteString(fmt.Sprintf("   %s  %s\n",
					tStyleMuted.Render("WHY:     "),
					tStyleBright.Render(truncate(issue.RootCause, 90)),
				))
			}

			// ACTION / LOOK FOR
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

			// FIX / CHECK command
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

				// copy hint
				copyWidth := tWidth() - 20
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

			// RISK
			if issue.Risk != "" {
				inner.WriteString(fmt.Sprintf("   %s  %s\n",
					tStyleMuted.Render("RISK:    "),
					tStyleWarn.Render(truncate(issue.Risk, 90)),
				))
			}

			// CONFIDENCE
			if issue.Confidence != "" {
				inner.WriteString(fmt.Sprintf("   %s  %s\n",
					tStyleMuted.Render("CONFIDENCE:"),
					confidenceStyle(issue.Confidence).Render(issue.Confidence),
				))
			}

			if i < len(m.issues)-1 {
				inner.WriteString(tStyleDivider.Render(strings.Repeat("─", tWidth()-10)) + "\n")
			}
		}
	}

	b.WriteString(tStyleBoxLLM.Render(inner.String()))
	b.WriteString("\n")

	// footer
	footerLeft := tStyleMuted.Render("press ") +
		tStyleBlue.Render("'esc'") +
		tStyleMuted.Render(" to return  ·  ") +
		tStyleBlue.Render("'r'") +
		tStyleMuted.Render(" to re-analyze  ·  ") +
		tStyleWarn.Render("1-9") +
		tStyleMuted.Render(" to copy fix command")

	footerRight := tStyleMuted.Render(fmt.Sprintf("probe #%d  ·  ", m.probeCount)) +
		tStyleOk.Render(fmt.Sprintf("%d issues", len(m.issues)))

	fw := tWidth()
	frw := 30
	flw := fw - frw

	flb := lipgloss.NewStyle().Width(flw).Render(footerLeft)
	frb := lipgloss.NewStyle().Width(frw).Align(lipgloss.Right).Render(footerRight)

	b.WriteString(tDivider())
	b.WriteString("\n")
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, flb, frb))
	b.WriteString("\n")

	return b.String()
}

func renderTUIHeader(m Model) string {
	brand := tStyleBright.Render("▸  S T E E R E D")
	tagline := lipgloss.NewStyle().Foreground(tColorMuted).Italic(true).Render("the light that guides you through darkness")

	clusterName := tStyleBlue.Render(m.client.ClusterName)
	ctx := tStyleMuted.Render(m.client.Context)
	ts := tStyleMuted.Render(time.Now().UTC().Format("2006-01-02  15:04:05 UTC"))

	left := brand + "\n" + tagline
	right := fmt.Sprintf("cluster: %s\ncontext: %s\n%s", clusterName, ctx, ts)

	w := tWidth() - 8
	rw := 50
	lw := w - rw

	lb := lipgloss.NewStyle().Width(lw).Render(left)
	rb := lipgloss.NewStyle().Width(rw).Align(lipgloss.Right).Render(right)

	content := lipgloss.JoinHorizontal(lipgloss.Top, lb, rb)
	return tStyleBoxHeader.Render(content)
}

func renderLiveBar(m Model) string {
	var dot string
	if m.nextProbe%2 == 0 {
		dot = tStyleOk.Render("●")
	} else {
		dot = tStyleMuted.Render("●")
	}

	liveLabel := tStyleOk.Render("LIVE")
	interval := tStyleMuted.Render("probing every ") + tStyleBlue.Render("30s")

	healthPct := 100
	if len(m.issues) > 0 && m.snapshot != nil {
		total := len(m.snapshot.Nodes) + len(m.snapshot.Deployments) + len(m.snapshot.Pods)
		if total > 0 {
			healthPct = 100 - (len(m.issues) * 100 / total)
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
		healthState = tStyleErr.Render(fmt.Sprintf("  ·  %d issues found  ", len(m.issues))) +
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

	w := tWidth() - 8
	rw := 45
	lw := w - rw

	lb := lipgloss.NewStyle().Width(lw).Render(left)
	rb := lipgloss.NewStyle().Width(rw).Align(lipgloss.Right).Render(right)

	content := lipgloss.JoinHorizontal(lipgloss.Top, lb, rb)
	return tStyleBoxLive.Render(content)
}

func renderHealthGrid(m Model) string {
	if m.snapshot == nil {
		return tStyleBoxHealth.Render(tStyleMuted.Render("  waiting for first probe..."))
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
		{fmt.Sprintf("%d", len(m.issues)), "issues", fmt.Sprintf("%d analyzing", len(m.analyzing)), tStyleWarn},
	}

	colW := (tWidth() - 8) / 4

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

	divider := tStyleDivider.Render(strings.Repeat("─", tWidth()-6))

	var health strings.Builder
	health.WriteString(buildRow(cells[:4]) + "\n")
	health.WriteString(divider + "\n")
	health.WriteString(buildRow(cells[4:]))

	return tStyleBoxHealth.Render(health.String())
}

func renderResolved(m Model) string {
	if len(m.resolved) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n")

	for _, r := range m.resolved {
		line := tStyleOk.Render("✓  "+r.Title) +
			tStyleMuted.Render("  —  fixed "+r.ResolvedAt.Format("15:04:05"))
		b.WriteString(lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3fb95033")).
			Padding(0, 1).
			Width(tWidth() - 4).
			Render(line))
		b.WriteString("\n")
	}

	return b.String()
}

func renderTUIFooter(m Model) string {
	left := tStyleMuted.Render("steered v2.0.0  ·  github.com/steereddev/steered  ·  ") +
		tStyleBlue.Render("ctrl+c") +
		tStyleMuted.Render(" to exit")

	right := tStyleMuted.Render(fmt.Sprintf("probe #%d  ·  collected in ", m.probeCount)) +
		tStyleOk.Render(fmt.Sprintf("%dms", m.probeTimeMs)) +
		tStyleMuted.Render(fmt.Sprintf("  ·  %d analyzing", len(m.analyzing)))

	w := tWidth()
	rw := 50
	lw := w - rw

	lb := lipgloss.NewStyle().Width(lw).Render(left)
	rb := lipgloss.NewStyle().Width(rw).Align(lipgloss.Center).Render(right)

	return lipgloss.JoinHorizontal(lipgloss.Top, lb, rb) + "\n"
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
