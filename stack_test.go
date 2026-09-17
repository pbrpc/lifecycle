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

	if err := stack.Unwind(t.Context()); err != nil {
		t.Fatalf("Cleanup() error = %v, want nil", err)
	}
}

func TestStackCleanup(t *testing.T) {
	ctx := t.Context()
	firstErr := errors.New("first cleanup")
	lastErr := errors.New("last cleanup")
	order := make([]string, 0, 3)
	var stack lifecycle.Stack

	stack.Push(nil)
	stack.Push(recordCleanup(t, ctx, &order, "first", firstErr))
	stack.Push(recordCleanup(t, ctx, &order, "second", nil))
	stack.Push(recordCleanup(t, ctx, &order, "last", lastErr))

	err := stack.Unwind(ctx)
	if !errors.Is(err, firstErr) {
		t.Errorf("Unwind() error = %v, want error wrapping %v", err, firstErr)
	}
	if !errors.Is(err, lastErr) {
		t.Errorf("Unwind() error = %v, want error wrapping %v", err, lastErr)
	}
	if want := []string{"last", "second", "first"}; !slices.Equal(order, want) {
		t.Errorf("Unwind() order = %v, want %v", order, want)
	}
}

func recordCleanup(
	t *testing.T,
	wantContext context.Context,
	order *[]string,
	name string,
	err error,
) lifecycle.CleanupFunc {
	t.Helper()

	return func(ctx context.Context) error {
		if ctx != wantContext {
			t.Errorf("cleanup context = %v, want test context %v", ctx, wantContext)
		}
		*order = append(*order, name)
		return err
	}
}
