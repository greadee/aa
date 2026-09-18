// Package replay builds a deterministic frame list from a session's events and
// provides a time-travel cursor over it.
//
// A frame is a projected graph plus its layout at one point in the sequence
// timeline. Frames are built by folding the events in Sequence order, one frame
// per event, so the cursor is a pure seek into a fixed list rather than a
// stateful re-simulation. Build is deterministic: the same events always yield
// the same frames, independent of delivery order.
package replay

import (
	"fmt"
	"sort"

	visualizer "github.com/greadee/aa/visualizer"
	"github.com/greadee/aa/visualizer/layout"
	"github.com/greadee/aa/visualizer/session"
)

// Frame is one deterministic snapshot of a session's timeline.
type Frame struct {
	Index     int                  `json:"index"`
	Sequence  int                  `json:"sequence"`
	EventID   string               `json:"eventId,omitempty"`
	Type      visualizer.EventType `json:"type,omitempty"`
	Graph     visualizer.Graph     `json:"graph"`
	Placement layout.Placement     `json:"placement"`
}

// FrameList is the ordered set of frames for one session.
type FrameList struct {
	SessionID visualizer.SessionID `json:"sessionId"`
	Frames    []Frame              `json:"frames"`
}

// Cursor returns a new cursor positioned at the first frame.
func (l FrameList) Cursor() (*Cursor, error) {
	return NewCursor(l.Frames)
}

// Build folds a session's events into one frame per event, in Sequence order.
//
// Each frame is produced by the same projection and layout path that live
// rendering uses, so replay never drifts from live. Events are ordered by
// Sequence, never the wall clock, so the same events always produce the same
// frame list regardless of the order they are delivered in.
func Build(sessionID visualizer.SessionID, events []visualizer.Event) (FrameList, error) {
	if sessionID == "" {
		return FrameList{}, fmt.Errorf("%w: sessionId is required", visualizer.ErrInvalid)
	}
	if len(events) == 0 {
		return FrameList{}, fmt.Errorf("%w: session %q", visualizer.ErrEmpty, sessionID)
	}
	ordered := orderedCopy(events)
	frames := make([]Frame, 0, len(ordered))
	for i, e := range ordered {
		graph, err := session.Project(sessionID, ordered[:i+1])
		if err != nil {
			return FrameList{}, err
		}
		placement, err := layout.Layout(graph)
		if err != nil {
			return FrameList{}, err
		}
		frames = append(frames, Frame{
			Index:     i,
			Sequence:  e.Sequence,
			EventID:   e.ID,
			Type:      e.Type,
			Graph:     graph,
			Placement: placement,
		})
	}
	return FrameList{SessionID: sessionID, Frames: frames}, nil
}

// Cursor is a time-travel position over a frame list.
type Cursor struct {
	frames []Frame
	index  int
}

// NewCursor returns a cursor positioned at the first frame.
func NewCursor(frames []Frame) (*Cursor, error) {
	if len(frames) == 0 {
		return nil, fmt.Errorf("%w: no frames", visualizer.ErrEmpty)
	}
	return &Cursor{frames: frames, index: 0}, nil
}

// Len returns the number of frames.
func (c *Cursor) Len() int { return len(c.frames) }

// Index returns the current frame index.
func (c *Cursor) Index() int { return c.index }

// AtStart reports whether the cursor is at the first frame.
func (c *Cursor) AtStart() bool { return c.index == 0 }

// AtEnd reports whether the cursor is at the last frame.
func (c *Cursor) AtEnd() bool { return len(c.frames) > 0 && c.index == len(c.frames)-1 }

// Snapshot returns the current frame.
func (c *Cursor) Snapshot() (Frame, error) {
	if err := c.available(); err != nil {
		return Frame{}, err
	}
	return c.frames[c.index], nil
}

// Step moves the cursor by delta frames and returns the new frame. When the
// target index is out of range the cursor is left where it was.
func (c *Cursor) Step(delta int) (Frame, error) {
	if err := c.available(); err != nil {
		return Frame{}, err
	}
	return c.Seek(c.index + delta)
}

// Seek moves the cursor to an absolute frame index and returns that frame. When
// the index is out of range the cursor is left where it was.
func (c *Cursor) Seek(index int) (Frame, error) {
	if err := c.available(); err != nil {
		return Frame{}, err
	}
	if index < 0 || index >= len(c.frames) {
		return Frame{}, fmt.Errorf("%w: frame index %d out of range [0,%d)", visualizer.ErrInvalid, index, len(c.frames))
	}
	c.index = index
	return c.frames[c.index], nil
}

// SeekSequence moves the cursor to the first frame at the given Sequence. A
// sequence that is not on the timeline returns ErrNotFound.
func (c *Cursor) SeekSequence(sequence int) (Frame, error) {
	if err := c.available(); err != nil {
		return Frame{}, err
	}
	for i, f := range c.frames {
		if f.Sequence == sequence {
			c.index = i
			return f, nil
		}
	}
	return Frame{}, fmt.Errorf("%w: sequence %d", visualizer.ErrNotFound, sequence)
}

// Reset moves the cursor to the first frame.
func (c *Cursor) Reset() (Frame, error) {
	return c.Seek(0)
}

func (c *Cursor) available() error {
	if c == nil || len(c.frames) == 0 {
		return fmt.Errorf("%w: no frames", visualizer.ErrEmpty)
	}
	return nil
}

func orderedCopy(events []visualizer.Event) []visualizer.Event {
	out := append([]visualizer.Event(nil), events...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Sequence < out[j].Sequence })
	return out
}
