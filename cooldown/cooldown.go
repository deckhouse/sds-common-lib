package cooldown

import "context"

// Represents a resource, which should not be used too frequently and needs
// "cooling" before next usage.
type Cooldown interface {
	// If not in cooldown, returns immediately, and starts the cooldown.
	// If in cooldown, waits until cooled and returns nil. Next delay will be
	// doubled, but not longer then maxDelay.
	// The only possible error is ctx.Err().
	Hit(ctx context.Context) error
}
