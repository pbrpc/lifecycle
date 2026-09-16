//revive:disable:package-comments
package lifecycle

import (
	"context"
	"errors"
)

// Stack collects shutdown functions and invokes them in reverse registration
// order. Its zero value is ready to use.
type Stack struct {
	shutdowns []ShutdownFunc
}

// Push adds shutdownFunc to the stack. Push ignores a nil shutdown function.
func (s *Stack) Push(shutdownFunc ShutdownFunc) {
	if shutdownFunc == nil {
		return
	}

	s.shutdowns = append(s.shutdowns, shutdownFunc)
}

// Shutdown invokes every registered shutdown function in reverse registration
// order with ctx. It continues after errors and joins all errors it encounters.
func (s *Stack) Shutdown(ctx context.Context) error {
	var errs []error
	for index := len(s.shutdowns) - 1; index >= 0; index-- {
		if err := s.shutdowns[index](ctx); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
