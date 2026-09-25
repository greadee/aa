package source

import (
	"fmt"

	"github.com/greadee/aa/obsv/transport"
	visualizer "github.com/greadee/aa/visualizer"
)

// DefaultOwner identifies the in-process obsv transport the visualizer owns
// when a config does not name one.
const DefaultOwner = "visualizer"

// Mode selects the obsv transport backing a source.
type Mode string

const (
	// ModeInProcess uses the in-process attach-or-own obsv transport.
	ModeInProcess Mode = "in_process"
	// ModeSocket uses the kernel host's obsv socket. The socket client is owned
	// by ISS-OBSV-1; until it lands, opening this mode reports ErrUnavailable.
	ModeSocket Mode = "socket"
)

// Config selects and configures the obsv transport behind an EventSource. It is
// the single entry point callers use, so the kernel-host socket can replace the
// in-process transport later without changing callers.
type Config struct {
	// Mode selects the transport. Empty means ModeInProcess.
	Mode Mode
	// Owner identifies the in-process transport owner. Empty means DefaultOwner.
	Owner string
	// Socket is the kernel-host obsv socket address used by ModeSocket.
	Socket string
	// NewJournal supplies the in-process journal constructor. Nil uses an
	// in-memory journal per session.
	NewJournal transport.NewJournal
	// Transport overrides Mode with an explicit transport. A caller can supply
	// the kernel-host socket client here, or a durable in-process transport,
	// without changing this API.
	Transport transport.Server
}

// Open returns the EventSource selected by cfg.
func Open(cfg Config) (*OBsv, error) {
	if cfg.Transport != nil {
		return New(cfg.Transport)
	}
	switch cfg.Mode {
	case "", ModeInProcess:
		owner := cfg.Owner
		if owner == "" {
			owner = DefaultOwner
		}
		return NewLocal(owner, cfg.NewJournal), nil
	case ModeSocket:
		return nil, fmt.Errorf("%w: kernel obsv socket %q is not available yet", visualizer.ErrUnavailable, cfg.Socket)
	default:
		return nil, fmt.Errorf("%w: unknown source mode %q", visualizer.ErrInvalid, cfg.Mode)
	}
}
