// Package retention trims a projected session trail deterministically.
//
// Retention is applied to the visualizer's projected trail, never to the
// canonical event source: aa-memory remains authoritative and the visualizer
// only bounds its own view. Trimming keeps the most recent frames, drops frames
// below a sequence floor, and re-indexes what remains.
package retention

import "github.com/greadee/aa/visualizer/replay"

// Policy bounds a projected trail.
type Policy struct {
	// MaxFrames keeps at most this many most-recent frames. Zero or less means
	// no frame limit.
	MaxFrames int
	// MinSequence drops frames with Sequence below this floor. Zero or less
	// means no floor.
	MinSequence int
}

// DefaultPolicy retains every frame.
func DefaultPolicy() Policy { return Policy{} }

// Trim returns a new, re-indexed trail with the policy applied. The input and
// the frames it points at are never modified.
func Trim(frames []replay.Frame, p Policy) []replay.Frame {
	kept := make([]replay.Frame, 0, len(frames))
	for _, f := range frames {
		if p.MinSequence > 0 && f.Sequence < p.MinSequence {
			continue
		}
		kept = append(kept, f)
	}
	if p.MaxFrames > 0 && len(kept) > p.MaxFrames {
		kept = kept[len(kept)-p.MaxFrames:]
	}
	out := make([]replay.Frame, len(kept))
	for i, f := range kept {
		f.Index = i
		out[i] = f
	}
	return out
}

// TrimList applies the policy to a frame list and returns a new list with the
// same session id.
func TrimList(list replay.FrameList, p Policy) replay.FrameList {
	return replay.FrameList{SessionID: list.SessionID, Frames: Trim(list.Frames, p)}
}
