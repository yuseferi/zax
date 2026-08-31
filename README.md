<div align="center">

# ⚡ Zax

### Context-Aware Logging for Go with Uber's Zap

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.26.1-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![Go Reference](https://pkg.go.dev/badge/github.com/yuseferi/zax/v2.svg)](https://pkg.go.dev/github.com/yuseferi/zax/v2)
[![codecov](https://img.shields.io/codecov/c/github/yuseferi/zax?style=flat-square&logo=codecov)](https://codecov.io/github/yuseferi/zax)
[![Go Report Card](https://goreportcard.com/badge/github.com/yuseferi/zax?style=flat-square)](https://goreportcard.com/report/github.com/yuseferi/zax)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg?style=flat-square)](https://www.gnu.org/licenses/agpl-3.0)
[![GitHub release](https://img.shields.io/github/v/release/yuseferi/zax?style=flat-square&logo=github)](https://github.com/yuseferi/zax/releases)

[![CodeQL](https://github.com/yuseferi/zax/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/yuseferi/zax/actions/workflows/github-code-scanning/codeql)
[![Check & Build](https://github.com/yuseferi/zax/actions/workflows/ci.yml/badge.svg)](https://github.com/yuseferi/zax/actions/workflows/ci.yml)

<br />

**Zax** seamlessly integrates [Zap Logger](https://github.com/uber-go/zap) with Go's `context.Context`, enabling you to carry structured logging fields across your entire request lifecycle without boilerplate.

</div>

## 📚 Table of Contents

- [Why Zax?](#-why-zax)
- [Features](#-features)
- [Installation](#-installation)
- [Tasks](#-tasks)
- [Releases](#-releases)
- [Quick Start](#-quick-start)
- [API Reference](#-api-reference)
- [Integrations](#-integrations)
- [Real-World Example](#-real-world-example)
- [Benchmarks](#-benchmarks)
- [Contributing](#-contributing)
- [License](#-license)

---

## 🎯 Why Zax?

In modern Go applications, especially microservices, you often need to:

- 🔍 **Trace requests** across multiple functions and services
- 📊 **Correlate logs** with trace IDs, span IDs, and user context
- 🧹 **Avoid boilerplate** by not passing loggers as function parameters
- ⚡ **Maintain performance** without sacrificing structured logging

Zax solves these problems elegantly by storing Zap fields in context, making them available wherever you need to log.

## ✨ Features

| Feature | Description |
|---------|-------------|
| 🚀 **Zero Dependencies** | The core package only requires `go.uber.org/zap` |
| 🎯 **Context-Native** | Works seamlessly with Go's `context.Context` |
| ⚡ **High Performance** | Minimal, predictable overhead (see [Benchmarks](#-benchmarks)) |
| 🔧 **Simple API** | Just 7 functions to learn |
| 🔌 **First-Party Integrations** | HTTP middleware, gRPC interceptors, and OpenTelemetry trace correlation (see [Integrations](#-integrations)) |
| 🍬 **SugaredLogger Support** | Works with both `*zap.Logger` and `*zap.SugaredLogger` |
| 🧪 **Well Tested** | Comprehensive test coverage |

## 📦 Installation

```bash
go get -u github.com/yuseferi/zax/v2
```

**Requirements:** Go 1.26.1 or higher

## 🛠 Tasks

This project uses [Task](https://taskfile.dev/) to keep local commands and CI in sync.

Install Task:

```bash
go run github.com/go-task/task/v3/cmd/task@latest --version
```

Common commands:

```bash
task build
task test
task test:race
task lint
task bench
task ci
```

`task ci` runs the same lint and test flow used by GitHub Actions.

## 🚀 Releases

This project uses [semantic-release](https://github.com/semantic-release/semantic-release) to automate Git tags and GitHub releases.

Release automation is configured in `.releaserc.json` and runs from [.github/workflows/release.yml](.github/workflows/release.yml) after the `Quality check` workflow succeeds on `master`.

Use Conventional Commits so semantic-release can determine the next version:

- `fix:` for patch releases
- `feat:` for minor releases
- `feat!:` or a `BREAKING CHANGE:` footer for major releases

You can preview the next release locally with:

```bash
task release:dry-run
```

## 🚀 Quick Start

```go
package main

import (
    "context"
    
    "github.com/yuseferi/zax/v2"
    "go.uber.org/zap"
)

func main() {
    logger, _ := zap.NewProduction()
    defer logger.Sync()
    
    ctx := context.Background()
    
    // Add trace_id to context
    ctx = zax.Set(ctx, []zap.Field{
        zap.String("trace_id", "abc-123"),
        zap.String("user_id", "user-456"),
    })
    
    // Log with context fields - automatically includes trace_id and user_id
    logger.With(zax.Get(ctx)...).Info("request started")
    
    // Pass context to other functions
    processRequest(ctx, logger)
}

func processRequest(ctx context.Context, logger *zap.Logger) {
    // All logs automatically include trace_id and user_id!
    logger.With(zax.Get(ctx)...).Info("processing request")
    
    // Append additional fields without losing existing ones
    ctx = zax.Append(ctx, []zap.Field{
        zap.String("step", "validation"),
    })
    
    logger.With(zax.Get(ctx)...).Info("validation complete")
}
```

**Output:**
```json
{"level":"info","msg":"request started","trace_id":"abc-123","user_id":"user-456"}
{"level":"info","msg":"processing request","trace_id":"abc-123","user_id":"user-456"}
{"level":"info","msg":"validation complete","trace_id":"abc-123","user_id":"user-456","step":"validation"}
```

## 📖 API Reference

### Core Functions

#### `Set(ctx, fields) context.Context`
Stores zap fields in context. **Replaces** any existing fields.

```go
ctx = zax.Set(ctx, []zap.Field{
    zap.String("trace_id", "my-trace-id"),
    zap.Int("request_num", 42),
})
```

#### `Append(ctx, fields) context.Context`
Appends fields to existing context fields. **Preserves** previously set fields.

```go
// Existing: trace_id
ctx = zax.Append(ctx, []zap.Field{
    zap.String("span_id", "my-span-id"),
})
// Now has: trace_id + span_id
```

When the same key is added multiple times, later fields follow Zap's normal behavior and take precedence at log time.

#### `Get(ctx) []zap.Field`
Retrieves all stored fields from context.

```go
fields := zax.Get(ctx)
logger.With(fields...).Info("message")
```

#### `GetField(ctx, key) zap.Field`
Retrieves a specific field by key. Returns the zero-value `zap.Field` when the key is not present.

```go
traceField := zax.GetField(ctx, "trace_id")
fmt.Println(traceField.String) // "my-trace-id"
```

#### `LookupField(ctx, key) (zap.Field, bool)`
Like `GetField`, but also reports whether the key was found — useful when a
stored field may legitimately hold a zero value.

```go
if field, ok := zax.LookupField(ctx, "attempt"); ok {
    fmt.Println(field.Integer)
}
```

#### `Remove(ctx, keys...) context.Context`
Returns a context with the given keys removed from the stored fields.
Handy for scrubbing sensitive data (e.g. PII) before passing a context on.

```go
ctx = zax.Remove(ctx, "user_email", "api_token")
```

#### `GetSugared(ctx) []any`
Returns fields as key-value pairs for `SugaredLogger`.

```go
sugar := logger.Sugar()
sugar.With(zax.GetSugared(ctx)...).Info("sugared log")
```

`GetSugared` converts fields through Zap's encoder so common field types like strings, bools, numbers, errors, durations, and times are preserved.

## 🔌 Integrations

First-party integrations live in sub-packages of the same module, so you can adopt them with one line each.

### HTTP Middleware (`zaxhttp`)

Enriches every request with `request_id`, `method`, `path`, and `remote_addr` fields, and logs request start (Debug) and completion (Info/Warn/Error by status) with those fields.

```go
import "github.com/yuseferi/zax/v2/zaxhttp"

mux.Use(zaxhttp.Middleware(logger))

// Optional: customize request ID propagation
mux.Use(zaxhttp.Middleware(logger,
    zaxhttp.WithRequestIDHeader("X-Trace-ID"),
    zaxhttp.WithRequestIDGenerator(myGenerator),
))
```

The request ID is read from the `X-Request-ID` header (configurable) or generated (16 random hex bytes by default). Handlers see the enriched context through `r.Context()`.

### gRPC Interceptors (`zaxgrpc`)

Unary and stream server interceptors that pull a request ID from incoming metadata (`x-request-id` or `request-id`, configurable) and log each RPC with `grpc_method`, `grpc_code`, and `duration`.

```go
import "github.com/yuseferi/zax/v2/zaxgrpc"

server := grpc.NewServer(
    grpc.UnaryInterceptor(zaxgrpc.UnaryServerInterceptor(logger)),
    grpc.StreamInterceptor(zaxgrpc.StreamServerInterceptor(logger)),
)
```

### OpenTelemetry Trace Correlation (`zaxotel`)

Adds the active span's `trace_id`, `span_id`, and `sampled` flag to your zax fields, so logs link straight to traces. Only `go.opentelemetry.io/otel/trace` is required — no SDK dependency.

```go
import "github.com/yuseferi/zax/v2/zaxotel"

ctx = zaxotel.WithTrace(ctx)

// Or get the fields directly
logger.Info("processed", append(zax.Get(ctx), zaxotel.Fields(ctx)...)...)
```

## 🔥 Real-World Example

### HTTP Middleware with Distributed Tracing

```go
package main

import (
    "context"
    "net/http"
    
    "github.com/yuseferi/zax/v2"
    "go.uber.org/zap"
)

type Server struct {
    logger *zap.Logger
}

// Middleware injects trace context into all requests
func (s *Server) TracingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        
        // Extract or generate trace ID
        traceID := r.Header.Get("X-Trace-ID")
        if traceID == "" {
            traceID = generateTraceID()
        }
        
        // Store in context
        ctx = zax.Set(ctx, []zap.Field{
            zap.String("trace_id", traceID),
            zap.String("method", r.Method),
            zap.String("path", r.URL.Path),
        })
        
        s.logger.With(zax.Get(ctx)...).Info("request received")
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Handler automatically has access to trace context
func (s *Server) HandleUser(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    // Add handler-specific context
    ctx = zax.Append(ctx, []zap.Field{
        zap.String("handler", "user"),
    })
    
    user, err := s.fetchUser(ctx)
    if err != nil {
        s.logger.With(zax.Get(ctx)...).Error("failed to fetch user", zap.Error(err))
        http.Error(w, "Internal Error", 500)
        return
    }
    
    s.logger.With(zax.Get(ctx)...).Info("user fetched successfully",
        zap.String("user_id", user.ID),
    )
}

func (s *Server) fetchUser(ctx context.Context) (*User, error) {
    // All logs here include trace_id, method, path, and handler!
    s.logger.With(zax.Get(ctx)...).Debug("querying database")
    // ... database logic
    return &User{}, nil
}
```

## 📊 Benchmarks

Zax V2 is optimized for performance. Here's how it compares:

Run benchmarks yourself with:

```bash
task bench
# or
go test -bench . -run '^$' -benchmem ./...
```

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| **Pure Zap** | ~43 | 128 | 1 |
| **Zax V2** | ~226 | 584 | 5 |

> 💡 The extra allocations come from defensively cloning fields on `Set`/`Append`/`Get`
> and from storing them in the context, so callers can never mutate fields after
> they are stored. If you need the absolute minimum overhead, pass zap fields directly.

<details>
<summary>📋 Full Benchmark Results</summary>

Measured on an Apple M2 Pro with Go 1.26 and zap v1.28.0:

```text
pkg: github.com/yuseferi/zax/v2
BenchmarkLoggingWithOnlyZap-10    84292609    43.15 ns/op    128 B/op    1 allocs/op
BenchmarkLoggingWithZax-10        15688470   226.2 ns/op    584 B/op    5 allocs/op
PASS
```

</details>

## 🤝 Contributing

We ❤️ contributions! Here's how you can help:

1. 🍴 **Fork** the repository
2. 🌿 **Create** a feature branch (`git checkout -b feature/amazing-feature`)
3. 💻 **Commit** your changes (`git commit -m 'Add amazing feature'`)
4. 📤 **Push** to the branch (`git push origin feature/amazing-feature`)
5. 🎉 **Open** a Pull Request

### Development

```bash
# Clone the repository
git clone https://github.com/yuseferi/zax.git
cd zax

# Run tests
go test -v ./...

# Run benchmarks
go test -bench=. -benchmem

# Run linter
golangci-lint run
```

## 📄 License

This project is licensed under the **GNU Affero General Public License v3.0** - see the [LICENSE](LICENSE) file for details.

---

<div align="center">

**Made with ❤️ by [Yusef Mohamadi](https://github.com/yuseferi) and contributors**

⭐ **Star this repo** if you find it useful!

[Report Bug](https://github.com/yuseferi/zax/issues) •
[Request Feature](https://github.com/yuseferi/zax/issues)

</div>
