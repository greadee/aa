// Package tui is the console's text surface: a line-oriented dashboard over the
// view model. It is stdlib-only and deterministic; the same view model drives
// the web surface.
package tui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/greadee/aa/ui/command"
	"github.com/greadee/aa/ui/controlplane"
	"github.com/greadee/aa/ui/state"
	"github.com/greadee/aa/ui/views"
)

// Surface is the text console.
type Surface struct {
	client    controlplane.Client
	projectID string
	model     state.Model
	audit     *command.Audit
}

// New returns a text surface over a client.
func New(client controlplane.Client, projectID string) *Surface {
	return &Surface{client: client, projectID: projectID, audit: command.NewAudit()}
}

// Refresh reloads the view model.
func (s *Surface) Refresh(ctx context.Context) error {
	m, err := state.Load(ctx, s.client, s.projectID)
	if err != nil {
		return err
	}
	m.Active = s.model.Active
	if m.Active == "" {
		m.Active = state.ViewProjects
	}
	m.Message = s.model.Message
	s.model = m
	return nil
}

// View returns the rendered dashboard.
func (s *Surface) View() string { return views.Render(s.model) }

const helpText = "commands: :open <view> :project <id> :submit <title> :approve <gate> :create-project <id> <name> :refresh :quit/exit; keys: j/k, arrows, Tab"

// Handle applies one input line (a single-key shortcut or a palette command)
// and reports whether to quit.
func (s *Surface) Handle(ctx context.Context, input string) bool {
	raw := strings.TrimRight(input, "\r\n")
	line := strings.TrimSpace(raw)

	// Line-oriented arrow-key and Tab handling. These sequences are submitted
	// with Enter in the current non-raw terminal mode.
	switch {
	case raw == "\t" || line == "\x1b[B" || line == "\x1b[C":
		s.model.Active = state.Cycle(s.model.Active, 1)
		return false
	case line == "\x1b[A" || line == "\x1b[D":
		s.model.Active = state.Cycle(s.model.Active, -1)
		return false
	}

	// Single-key shortcuts.
	switch line {
	case "", "r":
		_ = s.Refresh(ctx)
		return false
	case "q":
		return true
	case "j", "]":
		s.model.Active = state.Cycle(s.model.Active, 1)
		return false
	case "k", "[":
		s.model.Active = state.Cycle(s.model.Active, -1)
		return false
	case "?":
		s.model.Message = helpText
		return false
	}

	act, err := command.Parse(line)
	if err != nil {
		return false
	}
	switch act.Verb {
	case command.Quit, command.Exit:
		return true
	case command.Help:
		s.model.Message = helpText
	case command.Project:
		if len(act.Args) == 0 {
			s.model.Message = "project: " + s.projectID
			break
		}
		id := act.Args[0]
		if _, err := s.client.Project(ctx, id); err != nil {
			s.model.Message = "error: " + err.Error()
			break
		}
		s.projectID = id
		if err := s.Refresh(ctx); err != nil {
			s.model.Message = "error: " + err.Error()
		} else {
			s.model.Message = "project " + id
		}
	case command.Open:
		if len(act.Args) > 0 {
			s.model.Active = state.View(act.Args[0])
		}
	case command.Refresh:
		_ = s.Refresh(ctx)
	case command.Submit:
		if len(act.Args) > 0 && s.audit.Apply(act.Raw, act) {
			id, err := s.client.SubmitWork(ctx, s.projectID, strings.Join(act.Args, " "), nil)
			if err != nil {
				s.model.Message = "error: " + err.Error()
			} else {
				s.model.Message = "submitted " + id
			}
			_ = s.Refresh(ctx)
		}
	case command.Approve:
		if len(act.Args) > 0 && s.audit.Apply(act.Raw, act) {
			if err := s.client.Approve(ctx, act.Args[0]); err != nil {
				s.model.Message = "error: " + err.Error()
			} else {
				s.model.Message = "approved " + act.Args[0]
			}
			_ = s.Refresh(ctx)
		}
	case command.Create:
		if len(act.Args) >= 2 && s.audit.Apply(act.Raw, act) {
			if err := s.client.CreateProject(ctx, act.Args[0], strings.Join(act.Args[1:], " ")); err != nil {
				s.model.Message = "error: " + err.Error()
			} else {
				s.model.Message = "created " + act.Args[0]
			}
			_ = s.Refresh(ctx)
		}
	default:
		s.model.Message = "unknown command: " + act.Verb
	}
	return false
}

// Run drives a line-oriented loop over in/out until EOF or quit.
func (s *Surface) Run(ctx context.Context, in io.Reader, out io.Writer) error {
	if err := s.Refresh(ctx); err != nil {
		return err
	}
	sc := bufio.NewScanner(in)
	for {
		fmt.Fprintln(out, s.View())
		fmt.Fprint(out, "> ")
		if !sc.Scan() {
			return nil
		}
		if s.Handle(ctx, sc.Text()) {
			return nil
		}
	}
}

// Audit returns the applied-command log.
func (s *Surface) Audit() []command.Entry { return s.audit.Entries() }
