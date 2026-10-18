package visplane

import (
	"context"
	"testing"

	"github.com/greadee/aa/app/internal/obsplane"
	"github.com/greadee/aa/memory/projection"
	"github.com/greadee/aa/memory/retrieval"
	"github.com/greadee/aa/memory/store"
	"github.com/greadee/aa/obsv/protocol"
	visualizer "github.com/greadee/aa/visualizer"
	"github.com/greadee/aa/visualizer/retention"
)

func TestComposedVisualizerServesSessionsTimelineAndGraph(t *testing.T) {
	obs, err := obsplane.New(obsplane.Config{Owner: "test"})
	if err != nil {
		t.Fatal(err)
	}
	defer obs.Close()

	for i := 0; i < 2; i++ {
		if !obs.Emit("session-1", protocol.Event{
			SourceType:    protocol.SourceWorkDelta,
			Confidence:    protocol.ConfidenceExact,
			Action:        "work.step",
			WorkPackageID: "wp-1",
		}) {
			t.Fatal("emit was rejected")
		}
	}

	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.AppendEvent([]byte(`{"contractVersion":"2.0","kind":"event","id":"e1","sequence":0,"occurredAt":"1970-01-01T00:00:00Z","type":"telemetry.recorded","aggregate":{"kind":"work_package","id":"wp-c"},"actor":{"kind":"agent","id":"a1"},"payload":{"sessionId":"canonical-1"}}`)); err != nil {
		t.Fatal(err)
	}
	query := retrieval.New(projection.NewMem(), st)

	p, err := New(Config{
		Observations:     obs.Server(),
		ObservedSessions: obs.Sessions,
		Memory:           query,
		Retention:        retention.Policy{MaxFrames: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	sessions, err := p.Sessions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 || sessions[0].ID != "canonical-1" || sessions[1].ID != "session-1" {
		t.Fatalf("sessions = %+v", sessions)
	}

	timeline, err := p.Timeline(ctx, "session-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(timeline.Frames) != 1 || timeline.Frames[0].Index != 0 || timeline.Frames[0].Sequence != 1 {
		t.Fatalf("timeline = %+v", timeline)
	}

	graph, err := p.Graph(ctx, "session-1")
	if err != nil {
		t.Fatal(err)
	}
	if graph.SessionID != visualizer.SessionID("session-1") || len(graph.Nodes) < 2 {
		t.Fatalf("graph = %+v", graph)
	}
}

func TestUnknownObservationSessionIsEmpty(t *testing.T) {
	obs, err := obsplane.New(obsplane.Config{Owner: "test"})
	if err != nil {
		t.Fatal(err)
	}
	defer obs.Close()
	p, err := New(Config{Observations: obs.Server()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Timeline(context.Background(), "missing"); err == nil {
		t.Fatal("expected an empty-stream error")
	}
}
