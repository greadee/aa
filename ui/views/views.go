// Package views renders the console view model to text. Rendering is
// metadata-only and deterministic; no color is required for meaning.
package views

import (
	"fmt"
	"strings"

	"github.com/greadee/aa/ui/state"
)

const navWidth = 16

// Render returns the dashboard: nav column, main pane, and a status bar.
func Render(m state.Model) string {
	body := columns(navLines(m.Active), contentLines(m), navWidth)
	var b strings.Builder
	fmt.Fprintf(&b, "aa console \u203a %s\n\n", m.Active)
	for _, line := range body {
		b.WriteString(strings.TrimRight(line, " "))
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	fmt.Fprintf(&b, "projects=%d nodes=%d    j/k or arrows/Tab: views  :open  ?: help  q/exit: quit", m.Status.Projects, m.Status.Nodes)
	if m.Message != "" {
		fmt.Fprintf(&b, "    %s", m.Message)
	}
	return b.String()
}

// RenderPlain returns a line-oriented rendering for non-interactive output.
func RenderPlain(m state.Model) string {
	var b strings.Builder
	fmt.Fprintf(&b, "view: %s\n", m.Active)
	for _, line := range contentLines(m) {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func navLines(active state.View) []string {
	out := make([]string, 0, len(state.Nav))
	for _, v := range state.Nav {
		if v == active {
			out = append(out, "\u25b8 "+string(v))
		} else {
			out = append(out, "  "+string(v))
		}
	}
	return out
}

func contentLines(m state.Model) []string {
	switch m.Active {
	case state.ViewProjects:
		return projectsLines(m)
	case state.ViewGraph:
		return graphLines(m)
	case state.ViewMemory:
		return memoryLines(m)
	case state.ViewNodes:
		return nodesLines(m)
	case state.ViewTools:
		return toolsLines(m)
	case state.ViewReports:
		return reportsLines(m)
	case state.ViewApprovals:
		return approvalsLines(m)
	default:
		return []string{"(no view)"}
	}
}

func projectsLines(m state.Model) []string {
	lines := []string{"PROJECTS"}
	if len(m.Projects) == 0 {
		lines = append(lines, "  (none)")
	}
	for _, p := range m.Projects {
		name := p.Name
		if name == "" {
			name = p.ID
		}
		lines = append(lines, "  "+p.ID+"  "+name)
	}
	lines = append(lines, "", "WORK ("+m.ProjectID+")")
	if len(m.Work) == 0 {
		lines = append(lines, "  (none)")
	}
	for _, w := range m.Work {
		lines = append(lines, fmt.Sprintf("  %-8s %-10s %s", w.ID, w.State, w.Title))
	}
	return lines
}

func graphLines(m state.Model) []string {
	lines := []string{"GRAPH (" + m.ProjectID + ")"}
	if len(m.Work) == 0 {
		return append(lines, "  (no work packages)")
	}
	readiness := map[string]string{}
	for _, node := range m.Graph.Nodes {
		readiness[node.WorkPackageID] = node.Readiness
	}
	for _, w := range m.Work {
		deps := "-"
		if len(w.DependsOn) > 0 {
			deps = strings.Join(w.DependsOn, ",")
		}
		ready := readiness[w.ID]
		if ready == "" {
			ready = "-"
		}
		lines = append(lines, fmt.Sprintf("  %-8s %-10s %-10s deps=%s", w.ID, w.State, ready, deps))
	}
	lines = append(lines, "", "ASSIGNMENTS")
	if len(m.Assignments) == 0 {
		return append(lines, "  (none)")
	}
	for _, a := range m.Assignments {
		lines = append(lines, fmt.Sprintf("  %-8s wp=%-8s %s", a.ID, a.WorkPackageID, a.State))
	}
	return lines
}

func nodesLines(m state.Model) []string {
	lines := []string{"NODES"}
	if len(m.Nodes) == 0 {
		return append(lines, "  (none)")
	}
	for _, n := range m.Nodes {
		auth := ""
		if n.Authority {
			auth = " (authority)"
		}
		lines = append(lines, fmt.Sprintf("  %-10s %s%s", n.ID, n.State, auth))
	}
	return lines
}

func toolsLines(m state.Model) []string {
	lines := []string{"TOOLS"}
	if len(m.Tools) == 0 {
		return append(lines, "  (none)")
	}
	for _, t := range m.Tools {
		sb := ""
		if t.Sandbox {
			sb = " sandbox"
		}
		lines = append(lines, fmt.Sprintf("  %-14s %-8s%s", t.ID, t.Kind, sb))
	}
	return lines
}

func memoryLines(m state.Model) []string {
	lines := []string{"MEMORY (" + m.ProjectID + ")"}
	lines = append(lines, "", "ISSUES")
	if len(m.Issues) == 0 {
		lines = append(lines, "  (none)")
	}
	for _, issue := range m.Issues {
		status := issue.Status
		if issue.Severity != "" {
			status = status + " " + issue.Severity
		}
		lines = append(lines, fmt.Sprintf("  %-14s %-10s %s", issue.ID, status, issue.Title))
	}
	lines = append(lines, "", "STRATEGIES")
	if len(m.Strategies) == 0 {
		lines = append(lines, "  (none)")
	}
	for _, strategy := range m.Strategies {
		lines = append(lines, fmt.Sprintf("  %-14s %-10s %s", strategy.ID, strategy.Lifecycle, strategy.Title))
	}
	return lines
}

func reportsLines(m state.Model) []string {
	lines := []string{"REPORTS"}
	if len(m.Sessions) == 0 {
		return append(lines, "  (no sessions)")
	}
	lines = append(lines, "", "SESSIONS")
	for _, session := range m.Sessions {
		lines = append(lines, fmt.Sprintf("  %-16s events=%d project=%s", session.ID, session.EventCount, session.ProjectID))
	}
	if m.Report.SessionID != "" {
		lines = append(lines, "", "REPORT ("+m.Report.SessionID+")")
		lines = append(lines, fmt.Sprintf("  events=%d sequences=%d..%d", m.Report.Events, m.Report.FirstSequence, m.Report.LastSequence))
		for _, count := range m.Report.BySourceType {
			lines = append(lines, fmt.Sprintf("  source %-12s %d", count.Label, count.Count))
		}
		for _, count := range m.Report.ByConfidence {
			lines = append(lines, fmt.Sprintf("  confidence %-8s %d", count.Label, count.Count))
		}
		if len(m.Report.WorkPackages) > 0 {
			lines = append(lines, "  work packages: "+strings.Join(m.Report.WorkPackages, ", "))
		}
		if len(m.Report.Actors) > 0 {
			lines = append(lines, "  actors: "+strings.Join(m.Report.Actors, ", "))
		}
	}
	return lines
}

func approvalsLines(m state.Model) []string {
	lines := []string{"APPROVALS"}
	pending := 0
	for _, a := range m.Assignments {
		if strings.EqualFold(a.State, "AWAITING_APPROVAL") || strings.EqualFold(a.State, "BLOCKED") {
			lines = append(lines, "  "+a.ID+"  "+a.State)
			pending++
		}
	}
	if pending == 0 {
		lines = append(lines, "  (nothing awaiting approval)")
	}
	return lines
}

func columns(left, right []string, width int) []string {
	n := len(left)
	if len(right) > n {
		n = len(right)
	}
	out := make([]string, n)
	for i := 0; i < n; i++ {
		l := ""
		if i < len(left) {
			l = left[i]
		}
		r := ""
		if i < len(right) {
			r = right[i]
		}
		out[i] = fmt.Sprintf("%-*s  %s", width, l, r)
	}
	return out
}
