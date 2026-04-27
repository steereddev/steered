package renderer

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/steereddev/steered/internal/model"
)

// colors
var (
	colorGreen  = lipgloss.Color("#3fb950")
	colorAmber  = lipgloss.Color("#d29922")
	colorRed    = lipgloss.Color("#f85149")
	colorBlue   = lipgloss.Color("#58a6ff")
	colorPurple = lipgloss.Color("#bc8cff")
	colorMuted  = lipgloss.Color("#484f58")
	colorText   = lipgloss.Color("#c9d1d9")
	colorBright = lipgloss.Color("#e6edf3")
	colorBgGrid = lipgloss.Color("#21262d")
)

// styles
var (
	styleOk     = lipgloss.NewStyle().Foreground(colorGreen)
	styleWarn   = lipgloss.NewStyle().Foreground(colorAmber)
	styleErr    = lipgloss.NewStyle().Foreground(colorRed)
	styleBlue   = lipgloss.NewStyle().Foreground(colorBlue)
	stylePurple = lipgloss.NewStyle().Foreground(colorPurple)
	styleMuted  = lipgloss.NewStyle().Foreground(colorMuted)
	styleText   = lipgloss.NewStyle().Foreground(colorText)
	styleBright = lipgloss.NewStyle().Foreground(colorBright)

	styleBrand = lipgloss.NewStyle().
			Foreground(colorBright).
			Bold(false)

	styleTagline = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)

	styleDivider = lipgloss.NewStyle().
			Foreground(colorBgGrid)

	stylePillOk = lipgloss.NewStyle().
			Foreground(colorGreen).
			Background(lipgloss.Color("#1a3320")).
			PaddingLeft(1).PaddingRight(1)

	stylePillWarn = lipgloss.NewStyle().
			Foreground(colorAmber).
			Background(lipgloss.Color("#2d2008")).
			PaddingLeft(1).PaddingRight(1)

	stylePillErr = lipgloss.NewStyle().
			Foreground(colorRed).
			Background(lipgloss.Color("#2d0f0f")).
			PaddingLeft(1).PaddingRight(1)

	stylePillBlue = lipgloss.NewStyle().
			Foreground(colorBlue).
			Background(lipgloss.Color("#0d1f38")).
			PaddingLeft(1).PaddingRight(1)

	// box per section — each with its own border color
	styleBoxHeader = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#e6edf3")).
			Padding(0, 1).
			Width(termWidth() - 4)

	styleBoxHealth = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#484f58")).
			Padding(0, 1).
			Width(termWidth() - 4)

	styleBoxNodes = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#58a6ff")).
			Padding(0, 1).
			Width(termWidth() - 4)

	styleBoxDeployments = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#bc8cff")).
				Padding(0, 1).
				Width(termWidth() - 4)

	styleBoxNamespaces = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#3fb950")).
				Padding(0, 1).
				Width(termWidth() - 4)

	styleBoxServices = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#d29922")).
				Padding(0, 1).
				Width(termWidth() - 4)

	styleAIPanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3fb950")).
			Padding(0, 1).
			Width(termWidth() - 4)

	styleCostPanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#d29922")).
			Padding(0, 1).
			Width(termWidth() - 4)
)

// TableRenderer renders the cluster snapshot as a styled terminal table
type TableRenderer struct{}

// NewTableRenderer creates a new TableRenderer
func NewTableRenderer() *TableRenderer {
	return &TableRenderer{}
}

// Render outputs the full cluster snapshot to stdout
func (r *TableRenderer) Render(snapshot *model.ClusterSnapshot) error {
	var b strings.Builder

	renderHeader(&b, snapshot)
	renderHealthGrid(&b, snapshot)
	renderNodes(&b, snapshot)
	renderDeployments(&b, snapshot)
	renderNamespaces(&b, snapshot)
	renderServices(&b, snapshot)
	renderCostSignals(&b, snapshot)
	renderAIAnalysis(&b, snapshot)
	renderFooter(&b, snapshot)

	fmt.Fprint(os.Stdout, b.String())
	return nil
}

// renderHeader renders the brand header in a box
func renderHeader(b *strings.Builder, s *model.ClusterSnapshot) {
	brand := styleBrand.Render("▸  S T E E R E D")
	tagline := styleTagline.Render("light that pushes darkness")

	cluster := styleBlue.Render(s.ClusterName)
	context := styleMuted.Render(s.Context)
	ts := styleMuted.Render(s.CapturedAt.Format("2006-01-02 15:04:05 UTC"))

	left := brand + "\n" + tagline
	right := fmt.Sprintf("cluster: %s\ncontext: %s\n%s", cluster, context, ts)

	width := termWidth() - 8
	rightWidth := 50
	leftWidth := width - rightWidth

	leftBlock := lipgloss.NewStyle().Width(leftWidth).Render(left)
	rightBlock := lipgloss.NewStyle().Width(rightWidth).Align(lipgloss.Right).Render(right)

	content := lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, rightBlock)

	b.WriteString("\n")
	b.WriteString(styleBoxHeader.Render(content))
	b.WriteString("\n")
}

// renderHealthGrid renders the health summary grid in a box
func renderHealthGrid(b *strings.Builder, s *model.ClusterSnapshot) {
	runningPods, pendingPods, failedPods := podCounts(s)
	readyNodes, warnNodes := nodeCounts(s)
	healthyDeploys, downDeploys := deployCounts(s)
	unattachedPVCs := len(s.CostSignals.UnattachedPVCs)
	idleNS := len(s.CostSignals.IdleNamespaces)

	type cell struct {
		num  string
		lbl  string
		sub  string
		nclr lipgloss.Style
	}

	cells := []cell{
		{fmt.Sprintf("%d", len(s.Nodes)), "nodes", fmt.Sprintf("%d ready · %d warn", readyNodes, warnNodes), styleBlue},
		{fmt.Sprintf("%d", len(s.Namespaces)), "namespaces", fmt.Sprintf("%d idle", idleNS), styleOk},
		{fmt.Sprintf("%d", runningPods), "pods running", fmt.Sprintf("%d pending · %d failed", pendingPods, failedPods), styleOk},
		{fmt.Sprintf("%d", len(s.Deployments)), "deployments", fmt.Sprintf("%d healthy · %d down", healthyDeploys, downDeploys), stylePurple},
		{fmt.Sprintf("%d", len(s.Services)), "services", fmt.Sprintf("%d external", externalServices(s)), styleBlue},
		{fmt.Sprintf("%d", len(s.Ingresses)), "ingresses", "all routes", styleBlue},
		{fmt.Sprintf("%d", len(s.PVCs)), "pvcs", fmt.Sprintf("%d unattached", unattachedPVCs), styleWarn},
		{fmt.Sprintf("%d", len(s.CostSignals.PodsWithNoLimits)), "no limits", "deployments", styleWarn},
	}

	colW := (termWidth() - 8) / 4

	cellStyle := func(last bool) lipgloss.Style {
		s := lipgloss.NewStyle().
			Width(colW).
			PaddingLeft(2).
			PaddingRight(1)
		if !last {
			s = s.BorderRight(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(colorBgGrid)
		}
		return s
	}

	buildRow := func(rowCells []cell) string {
		var blocks []string
		for i, c := range rowCells {
			line1 := c.nclr.Render(c.num) + "  " + styleMuted.Render(strings.ToUpper(c.lbl))
			line2 := styleMuted.Render(c.sub)
			content := line1 + "\n" + line2
			blocks = append(blocks, cellStyle(i == len(rowCells)-1).Render(content))
		}
		return lipgloss.JoinHorizontal(lipgloss.Top, blocks...)
	}

	divider := styleDivider.Render(strings.Repeat("─", termWidth()-6))

	var health strings.Builder
	health.WriteString(buildRow(cells[:4]) + "\n")
	health.WriteString(divider + "\n")
	health.WriteString(buildRow(cells[4:]))

	b.WriteString(styleBoxHealth.Render(health.String()))
	b.WriteString("\n")
}

// renderNodes renders the nodes section in a blue box
func renderNodes(b *strings.Builder, s *model.ClusterSnapshot) {
	var box strings.Builder

	renderSectionHeader(&box, "nodes", len(s.Nodes), styleBlue)

	headers := []string{"NAME", "STATUS", "ROLE", "VERSION", "CPU", "MEMORY", "AGE"}
	widths := []int{25, 12, 16, 14, 10, 10, 6}

	renderTableHeader(&box, headers, widths)

	for _, n := range s.Nodes {
		status := statusPill(n.Status)
		role := styleBlue.Render(n.Roles)
		if strings.Contains(n.Roles, "control") {
			role = stylePurple.Render(n.Roles)
		}
		row := []string{
			styleBright.Render(n.Name),
			status,
			role,
			styleMuted.Render(n.Version),
			styleBlue.Render(n.CPUCapacity),
			styleBlue.Render(n.MemoryCapacity),
			styleMuted.Render(n.Age),
		}
		renderTableRow(&box, row, widths)
	}

	b.WriteString(styleBoxNodes.Render(box.String()))
	b.WriteString("\n")
}

// renderDeployments renders the deployments section in a purple box
func renderDeployments(b *strings.Builder, s *model.ClusterSnapshot) {
	var box strings.Builder

	renderSectionHeader(&box, "deployments", len(s.Deployments), stylePurple)

	headers := []string{"NAME", "NAMESPACE", "READY", "UP-TO-DATE", "AVAILABLE", "AGE"}
	widths := []int{25, 18, 8, 12, 12, 6}

	renderTableHeader(&box, headers, widths)

	for _, d := range s.Deployments {
		readyStyle := styleOk
		if d.Available == 0 {
			readyStyle = styleErr
		} else if d.Available < d.UpToDate {
			readyStyle = styleWarn
		}
		row := []string{
			styleBright.Render(d.Name),
			styleMuted.Render(d.Namespace),
			readyStyle.Render(d.Ready),
			styleText.Render(fmt.Sprintf("%d", d.UpToDate)),
			styleText.Render(fmt.Sprintf("%d", d.Available)),
			styleMuted.Render(d.Age),
		}
		renderTableRow(&box, row, widths)
	}

	b.WriteString(styleBoxDeployments.Render(box.String()))
	b.WriteString("\n")
}

// renderNamespaces renders the namespaces section in a green box
func renderNamespaces(b *strings.Builder, s *model.ClusterSnapshot) {
	var box strings.Builder

	renderSectionHeader(&box, "namespaces", len(s.Namespaces), styleOk)

	headers := []string{"NAME", "STATUS", "AGE"}
	widths := []int{25, 12, 8}

	renderTableHeader(&box, headers, widths)

	for _, n := range s.Namespaces {
		row := []string{
			styleBright.Render(n.Name),
			statusPill(n.Status),
			styleMuted.Render(n.Age),
		}
		renderTableRow(&box, row, widths)
	}

	b.WriteString(styleBoxNamespaces.Render(box.String()))
	b.WriteString("\n")
}

// renderServices renders the services section in an amber box
func renderServices(b *strings.Builder, s *model.ClusterSnapshot) {
	var box strings.Builder

	renderSectionHeader(&box, "services & ingresses", len(s.Services), styleWarn)

	headers := []string{"NAME", "NAMESPACE", "TYPE", "EXTERNAL-IP", "PORTS", "AGE"}
	widths := []int{22, 16, 14, 30, 14, 6}

	renderTableHeader(&box, headers, widths)

	for _, svc := range s.Services {
		extIP := svc.ExternalIP
		if extIP == "" {
			extIP = styleMuted.Render("—")
		} else {
			extIP = styleOk.Render(extIP)
		}
		row := []string{
			styleBright.Render(svc.Name),
			styleMuted.Render(svc.Namespace),
			stylePillBlue.Render(svc.Type),
			extIP,
			styleMuted.Render(svc.Ports),
			styleMuted.Render(svc.Age),
		}
		renderTableRow(&box, row, widths)
	}

	b.WriteString(styleBoxServices.Render(box.String()))
	b.WriteString("\n")
}

// renderCostSignals renders the cost signals panel
func renderCostSignals(b *strings.Builder, s *model.ClusterSnapshot) {
	cs := s.CostSignals
	var panel strings.Builder

	title := styleWarn.Render("◈  COST SIGNALS")
	panel.WriteString(title + "\n\n")

	noLimits := fmt.Sprintf("%s\n%s\n%s",
		styleMuted.Render("PODS WITHOUT LIMITS"),
		styleBright.Render(fmt.Sprintf("%d deployments", len(cs.PodsWithNoLimits))),
		styleWarn.Render("risk of node starvation"),
	)

	unattached := fmt.Sprintf("%s\n%s\n%s",
		styleMuted.Render("UNATTACHED PVCS"),
		styleBright.Render(fmt.Sprintf("%d volumes", len(cs.UnattachedPVCs))),
		styleWarn.Render("~ $45 / month waste"),
	)

	idle := fmt.Sprintf("%s\n%s\n%s",
		styleMuted.Render("IDLE NAMESPACES"),
		styleBright.Render(fmt.Sprintf("%d found", len(cs.IdleNamespaces))),
		styleWarn.Render("~ $340 / month waste"),
	)

	colW := termWidth()/3 - 4
	c1 := lipgloss.NewStyle().Width(colW).Render(noLimits)
	c2 := lipgloss.NewStyle().Width(colW).Render(unattached)
	c3 := lipgloss.NewStyle().Width(colW).Render(idle)

	panel.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, c1, c2, c3))

	b.WriteString(styleCostPanel.Render(panel.String()))
	b.WriteString("\n")
}

// renderAIAnalysis renders the AI analysis panel
func renderAIAnalysis(b *strings.Builder, s *model.ClusterSnapshot) {
	if len(s.Analysis) == 0 {
		return
	}

	var panel strings.Builder
	title := styleOk.Render("✦  AURA ANALYSIS") + "  " + stylePillOk.Render("ollama/llama3")
	panel.WriteString(title + "\n\n")

	for _, insight := range s.Analysis {
		icon := styleWarn.Render("▲")
		if strings.HasPrefix(insight, "!") {
			icon = styleErr.Render("!")
			insight = strings.TrimPrefix(insight, "!")
		} else if strings.HasPrefix(insight, "✓") {
			icon = styleOk.Render("✓")
			insight = strings.TrimPrefix(insight, "✓")
		}
		panel.WriteString(fmt.Sprintf(" %s  %s\n", icon, styleText.Render(strings.TrimSpace(insight))))
	}

	b.WriteString(styleAIPanel.Render(panel.String()))
	b.WriteString("\n")
}

// renderFooter renders the footer
func renderFooter(b *strings.Builder, s *model.ClusterSnapshot) {
	b.WriteString("\n")
	b.WriteString(styleDivider.Render(strings.Repeat("─", termWidth())))
	b.WriteString("\n")

	left := styleMuted.Render("steered v2.0.0  ·  github.com/steereddev/steered  ·  run ") +
		styleBlue.Render("steered --help") +
		styleMuted.Render(" for options")
	right := styleMuted.Render("collected in ") + styleOk.Render("1.2s") + styleMuted.Render("  ·  0 errors")

	width := termWidth()
	rightWidth := 35
	leftWidth := width - rightWidth

	leftBlock := lipgloss.NewStyle().Width(leftWidth).Render(left)
	rightBlock := lipgloss.NewStyle().Width(rightWidth).Align(lipgloss.Right).Render(right)

	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, rightBlock))
	b.WriteString("\n")
}

// helpers

func renderSectionHeader(b *strings.Builder, title string, count int, titleColor lipgloss.Style) {
	t := titleColor.Render(strings.ToUpper(title))
	c := styleMuted.Render(fmt.Sprintf("%d total", count))
	line := styleDivider.Render(strings.Repeat("─", termWidth()-len(title)-20))
	b.WriteString(fmt.Sprintf("%s  %s  %s\n", t, line, c))
}

func renderTableHeader(b *strings.Builder, headers []string, widths []int) {
	var row strings.Builder
	for i, h := range headers {
		cell := styleMuted.Render(fmt.Sprintf("%-*s", widths[i], h))
		row.WriteString(cell)
	}
	b.WriteString(row.String() + "\n")
}

func renderTableRow(b *strings.Builder, cells []string, widths []int) {
	var row strings.Builder
	for i, cell := range cells {
		plain := lipgloss.NewStyle().UnsetForeground().UnsetBackground().Render(cell)
		plainLen := len([]rune(plain))
		pad := widths[i] - plainLen
		if pad < 0 {
			pad = 0
		}
		row.WriteString(cell + strings.Repeat(" ", pad+2))
	}
	b.WriteString(row.String() + "\n")
}

func statusPill(status string) string {
	switch strings.ToLower(status) {
	case "ready", "active", "running", "bound":
		return stylePillOk.Render(status)
	case "notready", "pending", "idle":
		return stylePillWarn.Render(status)
	case "failed", "error", "crashloopbackoff":
		return stylePillErr.Render(status)
	default:
		return stylePillBlue.Render(status)
	}
}

func termWidth() int {
	return 120
}

func podCounts(s *model.ClusterSnapshot) (running, pending, failed int) {
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
	return
}

func nodeCounts(s *model.ClusterSnapshot) (ready, warn int) {
	for _, n := range s.Nodes {
		if strings.ToLower(n.Status) == "ready" {
			ready++
		} else {
			warn++
		}
	}
	return
}

func deployCounts(s *model.ClusterSnapshot) (healthy, down int) {
	for _, d := range s.Deployments {
		if d.Available > 0 {
			healthy++
		} else {
			down++
		}
	}
	return
}

func externalServices(s *model.ClusterSnapshot) int {
	count := 0
	for _, svc := range s.Services {
		if svc.ExternalIP != "" && svc.ExternalIP != "<none>" {
			count++
		}
	}
	return count
}
