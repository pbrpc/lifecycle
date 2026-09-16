//revive:disable:package-comments
package lifecycle

import "context"

// ShutdownFunc gracefully shuts down a provider, flushing any pending data.
type ShutdownFunc func(ctx context.Context) error
