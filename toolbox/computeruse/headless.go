package computeruse

import (
	"context"
	"fmt"

	"github.com/greadee/aa/toolbox"
)

// Headless is a declared Driver for environments with no display backend. It is
// a real implementation, not a test double: it performs no OS interaction and
// fails every action closed with toolbox.ErrUnsupported, so a computer-use call
// is refused rather than silently doing nothing.
//
// A desktop backend (screen capture / input injection) is added by the phase
// that owns the desktop surface; until then, Headless is what the composition
// root wires so the provider is a live, reachable capability.
type Headless struct{}

// NewHeadless returns a Headless driver.
func NewHeadless() Headless { return Headless{} }

// Perform fails closed: there is no display backend to act on.
func (Headless) Perform(ctx context.Context, action Action) (Outcome, error) {
	if err := ctx.Err(); err != nil {
		return Outcome{}, err
	}
	return Outcome{}, fmt.Errorf("%w: computer-use %q requires a desktop backend", toolbox.ErrUnsupported, action.Kind)
}

// Close is a no-op.
func (Headless) Close() error { return nil }

var _ Driver = Headless{}
