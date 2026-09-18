package retention

import (
	"reflect"
	"testing"

	visualizer "github.com/greadee/aa/visualizer"
	"github.com/greadee/aa/visualizer/replay"
)

func trail(sequences ...int) []replay.Frame {
	frames := make([]replay.Frame, len(sequences))
	for i, seq := range sequences {
		frames[i] = replay.Frame{
			Index:     i,
			Sequence:  seq,
			EventID:   "e",
			Graph:     visualizer.Graph{SessionID: "s1"},
			Placement: nil,
		}
	}
	return frames
}

func sequences(frames []replay.Frame) []int {
	out := make([]int, len(frames))
	for i, f := range frames {
		out[i] = f.Sequence
	}
	return out
}

func TestTrimMaxFramesKeepsNewest(t *testing.T) {
	got := Trim(trail(1, 2, 3, 4, 5), Policy{MaxFrames: 2})
	if !reflect.DeepEqual(sequences(got), []int{4, 5}) {
		t.Fatalf("sequences = %v", sequences(got))
	}
	for i, f := range got {
		if f.Index != i {
			t.Fatalf("frame[%d].Index = %d, want %d", i, f.Index, i)
		}
	}
}

func TestTrimMinSequenceDropsOld(t *testing.T) {
	got := Trim(trail(1, 2, 3, 4, 5), Policy{MinSequence: 3})
	if !reflect.DeepEqual(sequences(got), []int{3, 4, 5}) {
		t.Fatalf("sequences = %v", sequences(got))
	}
}

func TestTrimCombined(t *testing.T) {
	got := Trim(trail(1, 2, 3, 4, 5, 6), Policy{MinSequence: 2, MaxFrames: 2})
	if !reflect.DeepEqual(sequences(got), []int{5, 6}) {
		t.Fatalf("sequences = %v", sequences(got))
	}
}

func TestTrimDoesNotMutateInput(t *testing.T) {
	original := trail(1, 2, 3, 4, 5)
	before := sequences(original)
	_ = Trim(original, Policy{MinSequence: 3, MaxFrames: 1})
	if !reflect.DeepEqual(sequences(original), before) {
		t.Fatalf("input mutated: %v", sequences(original))
	}
	for i, f := range original {
		if f.Index != i {
			t.Fatalf("input index %d = %d", i, f.Index)
		}
	}
}

func TestTrimNoLimitReturnsAll(t *testing.T) {
	got := Trim(trail(1, 2, 3), DefaultPolicy())
	if !reflect.DeepEqual(sequences(got), []int{1, 2, 3}) {
		t.Fatalf("sequences = %v", sequences(got))
	}
}

func TestTrimEmpty(t *testing.T) {
	got := Trim(nil, Policy{MaxFrames: 3, MinSequence: 1})
	if len(got) != 0 {
		t.Fatalf("len = %d, want 0", len(got))
	}
}

func TestTrimListKeepsSession(t *testing.T) {
	list := replay.FrameList{SessionID: "s1", Frames: trail(1, 2, 3)}
	got := TrimList(list, Policy{MaxFrames: 1})
	if got.SessionID != "s1" {
		t.Fatalf("session = %q", got.SessionID)
	}
	if !reflect.DeepEqual(sequences(got.Frames), []int{3}) {
		t.Fatalf("sequences = %v", sequences(got.Frames))
	}
}
