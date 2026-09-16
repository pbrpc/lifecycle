package lifecycle_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/pbrpc/lifecycle"
)

func TestStackZeroValue(t *testing.T) {
	var stack lifecycle.Stack

	if err := stack.Shutdown(t.Context()); err != nil {
		t.Fatalf("Shutdown() error = %v, want nil", err)
	}
}

func TestStackShutdown(t *testing.T) {
	ctx := t.Context()
	firstErr := errors.New("first shutdown")
	lastErr := errors.New("last shutdown")
	order := make([]string, 0, 3)
	var stack lifecycle.Stack

	stack.Push(nil)
	stack.Push(recordShutdown(t, ctx, &order, "first", firstErr))
	stack.Push(recordShutdown(t, ctx, &order, "second", nil))
	stack.Push(recordShutdown(t, ctx, &order, "last", lastErr))

	err := stack.Shutdown(ctx)
	if !errors.Is(err, firstErr) {
		t.Errorf("Shutdown() error = %v, want error wrapping %v", err, firstErr)
	}
	if !errors.Is(err, lastErr) {
		t.Errorf("Shutdown() error = %v, want error wrapping %v", err, lastErr)
	}
	if want := []string{"last", "second", "first"}; !slices.Equal(order, want) {
		t.Errorf("Shutdown() order = %v, want %v", order, want)
	}
}

func recordShutdown(
	t *testing.T,
	wantContext context.Context,
	order *[]string,
	name string,
	err error,
) lifecycle.ShutdownFunc {
	t.Helper()

	return func(ctx context.Context) error {
		if ctx != wantContext {
			t.Errorf("shutdown context = %v, want test context %v", ctx, wantContext)
		}
		*order = append(*order, name)
		return err
	}
}
