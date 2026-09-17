//revive:disable:package-comments
package lifecycle

import (
	"context"
	"errors"
	"slices"
)

// CleanupFunc gracefully drains a component of any pending work.
type CleanupFunc func(ctx context.Context) error

// Stack collects cleanup functions and invokes them in reverse registration
// order. Its zero value is ready to use.
type Stack struct {
	cleanups []CleanupFunc
}

// Push adds cleanupFunc to the stack. Push ignores a nil cleanup function.
func (s *Stack) Push(cleanupFunc CleanupFunc) {
	if cleanupFunc == nil {
		return
	}

	s.cleanups = append(s.cleanups, cleanupFunc)
}

// Unwind invokes every registered cleanup function in reverse registration
// order with ctx. It continues after errors and joins all errors it encounters.
func (s *Stack) Unwind(ctx context.Context) error {
	var errs []error
	for _, v := range slices.Backward(s.cleanups) {
		if err := v(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
