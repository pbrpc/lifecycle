# lifecycle

`lifecycle` defines transport-independent shutdown contracts for pbrpc
libraries and services. It does not own signals, timeouts, logging, or any
particular server or client implementation.

## Installation

```bash
go get github.com/pbrpc/lifecycle
```

## Usage

The zero value of `Stack` is ready to use. Add shutdown functions as components
start, then shut them down with the context owned by the application:

```go
var shutdowns lifecycle.Stack

shutdowns.Push(cache.Shutdown)
shutdowns.Push(telemetry.Shutdown)

if err := shutdowns.Shutdown(ctx); err != nil {
	return err
}
```

`Shutdown` invokes every registered function in last-in, first-out order and
passes each one the supplied context unchanged. A failure does not prevent the
remaining functions from running; all failures are returned as one joined
error. Pushing a nil function has no effect.
