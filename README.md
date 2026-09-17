# lifecycle

`lifecycle` coordinates transport-independent graceful unwinding and draining
for pbrpc libraries and services.

## Installation

```bash
go get github.com/pbrpc/lifecycle
```

## Usage

The zero value of `Stack` is ready to use. Add cleanup functions as components
start. Components are drained in last-in, first-out order, so push telemetry
before the server when telemetry must remain available while requests drain:

```go
var stack lifecycle.Stack

stack.Push(lifecycle.Logged(log, "telemetry", flush))
stack.Push(lifecycle.Logged(log, "server", server.HTTP.Shutdown))

defer lifecycle.HandleGracefulShutdown(ctx, log, &stack, cleanupTimeout)
```

`HandleGracefulShutdown` creates one cleanup deadline and invokes the stack.
`Logged` records each component as it begins and completes draining. A failure
does not prevent the remaining functions from running; the stack joins all
failures. Pushing a nil function has no effect.
