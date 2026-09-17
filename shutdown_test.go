package lifecycle_test

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"testing"
	"time"

	"github.com/pbrpc/lifecycle"
	mocksslog "github.com/pbrpc/testing/mocks/slog"
)

func TestLogged(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := t.Context()
		var records []slog.Record
		log := slog.New(mocksslog.NewCaptureHandler(&records))
		called := false
		shutdown := lifecycle.Logged(log, "server", func(got context.Context) error {
			called = true
			if got != ctx {
				t.Errorf("shutdown context = %v, want %v", got, ctx)
			}

			return nil
		})

		if err := shutdown(ctx); err != nil {
			t.Fatalf("shutdown error = %v, want nil", err)
		}
		if !called {
			t.Fatal("shutdown function was not called")
		}
		assertRecords(t, records, []recordExpectation{
			{message: "Draining component", level: slog.LevelInfo, component: "server"},
			{message: "Component drained", level: slog.LevelInfo, component: "server"},
		})
	})

	t.Run("error", func(t *testing.T) {
		ctx := t.Context()
		wantErr := errors.New("drain failed")
		var records []slog.Record
		log := slog.New(mocksslog.NewCaptureHandler(&records))
		shutdown := lifecycle.Logged(log, "telemetry", func(got context.Context) error {
			if got != ctx {
				t.Errorf("shutdown context = %v, want %v", got, ctx)
			}

			return wantErr
		})

		if err := shutdown(ctx); !errors.Is(err, wantErr) {
			t.Fatalf("shutdown error = %v, want %v", err, wantErr)
		}
		assertRecords(t, records, []recordExpectation{
			{message: "Draining component", level: slog.LevelInfo, component: "telemetry"},
			{
				message:   "Component drain incomplete",
				level:     slog.LevelWarn,
				component: "telemetry",
				err:       wantErr,
			},
		})
	})

	t.Run("nil shutdown", func(t *testing.T) {
		log := slog.New(slog.DiscardHandler)
		if shutdown := lifecycle.Logged(log, "unused", nil); shutdown != nil {
			t.Fatalf("Logged() = %v, want nil", shutdown)
		}
	})
}

func TestHandleGracefulShutdown(t *testing.T) {
	t.Run("sees components pushed after defer registration", func(t *testing.T) {
		called := false
		func() {
			var stack lifecycle.Stack
			defer lifecycle.HandleGracefulShutdown(
				t.Context(), slog.New(slog.DiscardHandler), &stack, time.Minute,
			)

			stack.Push(func(context.Context) error {
				called = true

				return nil
			})
		}()

		if !called {
			t.Fatal("component pushed after defer registration was not drained")
		}
	})

	t.Run("drains in stack order", func(t *testing.T) {
		var records []slog.Record
		log := slog.New(mocksslog.NewCaptureHandler(&records))
		var order []string
		var stack lifecycle.Stack
		stack.Push(lifecycle.Logged(log, "telemetry", func(ctx context.Context) error {
			if err := ctx.Err(); err != nil {
				t.Errorf("telemetry context error = %v, want nil", err)
			}
			order = append(order, "telemetry")

			return nil
		}))
		stack.Push(lifecycle.Logged(log, "server", func(ctx context.Context) error {
			if err := ctx.Err(); err != nil {
				t.Errorf("server context error = %v, want nil", err)
			}
			order = append(order, "server")

			return nil
		}))

		lifecycle.HandleGracefulShutdown(t.Context(), log, &stack, time.Minute)

		if want := []string{"server", "telemetry"}; !slices.Equal(order, want) {
			t.Errorf("shutdown order = %v, want %v", order, want)
		}
		assertRecords(t, records, []recordExpectation{
			{message: "Initiating graceful shutdown", level: slog.LevelInfo},
			{message: "Draining component", level: slog.LevelInfo, component: "server"},
			{message: "Component drained", level: slog.LevelInfo, component: "server"},
			{message: "Draining component", level: slog.LevelInfo, component: "telemetry"},
			{message: "Component drained", level: slog.LevelInfo, component: "telemetry"},
			{message: "Graceful shutdown complete", level: slog.LevelInfo},
		})
	})

	t.Run("reports joined errors after draining every component", func(t *testing.T) {
		firstErr := errors.New("first drain failed")
		lastErr := errors.New("last drain failed")
		var records []slog.Record
		log := slog.New(mocksslog.NewCaptureHandler(&records))
		var stack lifecycle.Stack
		stack.Push(func(context.Context) error { return firstErr })
		stack.Push(func(context.Context) error { return lastErr })

		lifecycle.HandleGracefulShutdown(t.Context(), log, &stack, 0)

		if len(records) != 2 {
			t.Fatalf("record count = %d, want 2", len(records))
		}
		if records[0].Message != "Initiating graceful shutdown" {
			t.Errorf("first message = %q, want %q", records[0].Message, "Initiating graceful shutdown")
		}
		if records[1].Message != "Graceful shutdown incomplete" {
			t.Errorf("last message = %q, want %q", records[1].Message, "Graceful shutdown incomplete")
		}
		gotErr, ok := recordValue(records[1], "error").(error)
		if !ok {
			t.Fatalf("error attribute = %v, want error", recordValue(records[1], "error"))
		}
		if !errors.Is(gotErr, firstErr) || !errors.Is(gotErr, lastErr) {
			t.Errorf("error attribute = %v, want joined errors %v and %v", gotErr, firstErr, lastErr)
		}
		if got := recordValue(records[1], "timeout"); got != time.Duration(0) {
			t.Errorf("timeout attribute = %v, want 0s", got)
		}
	})
}

type recordExpectation struct {
	message   string
	level     slog.Level
	component string
	err       error
}

func assertRecords(t *testing.T, records []slog.Record, wants []recordExpectation) {
	t.Helper()

	if len(records) != len(wants) {
		t.Fatalf("record count = %d, want %d", len(records), len(wants))
	}

	for index, want := range wants {
		got := records[index]
		if got.Message != want.message {
			t.Errorf("record %d message = %q, want %q", index, got.Message, want.message)
		}
		if got.Level != want.level {
			t.Errorf("record %d level = %v, want %v", index, got.Level, want.level)
		}
		if want.component != "" {
			if component := recordValue(got, "component"); component != want.component {
				t.Errorf("record %d component = %v, want %q", index, component, want.component)
			}
		}
		if want.err != nil {
			gotErr, ok := recordValue(got, "error").(error)
			if !ok || !errors.Is(gotErr, want.err) {
				t.Errorf("record %d error = %v, want %v", index, recordValue(got, "error"), want.err)
			}
		}
	}
}

func recordValue(record slog.Record, key string) any {
	var value any
	record.Attrs(func(attribute slog.Attr) bool {
		if attribute.Key == key {
			value = attribute.Value.Any()

			return false
		}

		return true
	})

	return value
}
