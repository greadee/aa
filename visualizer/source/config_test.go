package source

import (
	"context"
	"errors"
	"testing"

	"github.com/greadee/aa/obsv/protocol"
	"github.com/greadee/aa/obsv/transport"
	visualizer "github.com/greadee/aa/visualizer"
)

func TestOpenDefaultsToInProcess(t *testing.T) {
	src, err := Open(Config{})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	// A working in-process transport reports an empty (never-observed) session.
	if _, err := src.Replay(context.Background(), "s1"); !errors.Is(err, visualizer.ErrEmpty) {
		t.Fatalf("err = %v, want ErrEmpty", err)
	}
}

func TestOpenInProcessMode(t *testing.T) {
	src, err := Open(Config{Mode: ModeInProcess, Owner: "visualizer"})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := src.Replay(context.Background(), "s1"); !errors.Is(err, visualizer.ErrEmpty) {
		t.Fatalf("err = %v, want ErrEmpty", err)
	}
}

func TestOpenExplicitTransport(t *testing.T) {
	server := transport.NewLocal("test", nil)
	t.Cleanup(func() { _ = server.Close() })

	src, err := Open(Config{Transport: server})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	appendTo(t, server, "s1", obsEvent(protocol.SourceTool, "wp1"))
	got, err := src.Replay(context.Background(), "s1")
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("events = %d, want 1", len(got))
	}
}

func TestOpenSocketUnavailable(t *testing.T) {
	_, err := Open(Config{Mode: ModeSocket, Socket: "/run/aa/obsv.sock"})
	if !errors.Is(err, visualizer.ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}
}

func TestOpenUnknownMode(t *testing.T) {
	_, err := Open(Config{Mode: "carrier_pigeon"})
	if !errors.Is(err, visualizer.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}
