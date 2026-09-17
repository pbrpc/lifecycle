//revive:disable:package-comments
package lifecycle

import (
	"context"
	"log/slog"
	"time"
)

// Logged decorates cleanupFunc with lifecycle logs identified by component.
// It returns nil when cleanupFunc is nil.
func Logged(
	log *slog.Logger,
	component string,
	cleanupFunc CleanupFunc,
) CleanupFunc {
	if cleanupFunc == nil {
		return nil
	}

	return func(ctx context.Context) error {
		log.LogAttrs(ctx, slog.LevelInfo, "Draining component",
			slog.String("component", component))

		if err := cleanupFunc(ctx); err != nil {
			log.LogAttrs(ctx, slog.LevelWarn, "Component drain incomplete",
				slog.String("component", component),
				slog.Any("error", err),
			)

			return err
		}

		log.LogAttrs(ctx, slog.LevelInfo, "Component drained",
			slog.String("component", component))

		return nil
	}
}

// HandleGracefulShutdown drains every component in stack against one deadline
// built from cleanupTimeout. It logs any joined shutdown error after every
// component has been given an opportunity to drain.
func HandleGracefulShutdown(
	ctx context.Context,
	log *slog.Logger,
	stack *Stack,
	cleanupTimeout time.Duration,
) {
	log.LogAttrs(ctx, slog.LevelInfo, "Initiating graceful shutdown")

	shutdownCtx, cancel := context.WithTimeout(ctx, cleanupTimeout)
	defer cancel()

	if err := stack.Unwind(shutdownCtx); err != nil {
		log.LogAttrs(ctx, slog.LevelWarn, "Graceful shutdown incomplete",
			slog.Duration("timeout", cleanupTimeout),
			slog.Any("error", err),
		)

		return
	}

	log.LogAttrs(ctx, slog.LevelInfo, "Graceful shutdown complete")
}
