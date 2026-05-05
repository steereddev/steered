package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/steereddev/steered/internal/client"
	"github.com/steereddev/steered/internal/collector"
	"github.com/steereddev/steered/internal/model"
)

// probeDone is sent when a probe completes
type probeDone struct {
	snapshot   *model.ClusterSnapshot
	durationMs int64
	err        error
}

// tickMsg is sent every second for countdown
type tickMsg time.Time

// tick returns a command that sends a tickMsg every second
func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// probe runs all collectors and returns the snapshot
func probe(c *client.Client) tea.Cmd {
	return func() tea.Msg {
		start := time.Now()
		snapshot := model.NewSnapshot(c.ClusterName, c.Context)

		ctx := context.Background()

		collectors := []collector.Collector{
			collector.NewNodeCollector(c),
			collector.NewNamespaceCollector(c),
			collector.NewPodCollector(c),
			collector.NewDeploymentCollector(c),
			collector.NewDaemonSetCollector(c),
			collector.NewServiceCollector(c),
			collector.NewIngressCollector(c),
			collector.NewPVCCollector(c),
			collector.NewSecurityCollector(c),
		}

		for _, col := range collectors {
			if err := col.Collect(ctx, snapshot); err != nil {
				snapshot.Errors = append(snapshot.Errors, err.Error())
			}
		}

		return probeDone{
			snapshot:   snapshot,
			durationMs: time.Since(start).Milliseconds(),
		}
	}
}
