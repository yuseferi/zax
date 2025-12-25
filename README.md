<div align="center">

# ⚡ Zax

### Context-Aware Logging for Go with Uber's Zap

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.21-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![Go Reference](https://pkg.go.dev/badge/github.com/yuseferi/zax/v2.svg)](https://pkg.go.dev/github.com/yuseferi/zax/v2)
[![codecov](https://img.shields.io/codecov/c/github/yuseferi/zax?style=flat-square&logo=codecov)](https://codecov.io/github/yuseferi/zax)
[![Go Report Card](https://goreportcard.com/badge/github.com/yuseferi/zax?style=flat-square)](https://goreportcard.com/report/github.com/yuseferi/zax)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg?style=flat-square)](https://www.gnu.org/licenses/agpl-3.0)
[![GitHub release](https://img.shields.io/github/v/release/yuseferi/zax?style=flat-square&logo=github)](https://github.com/yuseferi/zax/releases)

[![CodeQL](https://github.com/yuseferi/zax/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/yuseferi/zax/actions/workflows/github-code-scanning/codeql)
[![Check & Build](https://github.com/yuseferi/zax/actions/workflows/ci.yml/badge.svg)](https://github.com/yuseferi/zax/actions/workflows/ci.yml)

<br />

**Zax** seamlessly integrates [Zap Logger](https://github.com/uber-go/zap) with Go's `context.Context`, enabling you to carry structured logging fields across your entire request lifecycle without boilerplate.

[Features](#-features) •
[Installation](#-installation) •
[Quick Start](#-quick-start) •
[API Reference](#-api-reference) •
[Benchmarks](#-benchmarks) •
[Contributing](#-contributing)

</div>

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
| 🚀 **Zero Dependencies** | Only requires `go.uber.org/zap` |
| 🎯 **Context-Native** | Works seamlessly with Go's `context.Context` |
| ⚡ **High Performance** | Minimal overhead (~20ns per operation) |
| 🔧 **Simple API** | Just 5 functions to learn |
| 🍬 **SugaredLogger Support** | Works with both `*zap.Logger` and `*zap.SugaredLogger` |
| 🧪 **Well Tested** | Comprehensive test coverage |

## 📦 Installation

```bash
go get -u github.com/yuseferi/zax/v2
```

**Requirements:** Go 1.21 or higher

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

#### `Get(ctx) []zap.Field`
Retrieves all stored fields from context.

```go
fields := zax.Get(ctx)
logger.With(fields...).Info("message")
```

#### `GetField(ctx, key) zap.Field`
Retrieves a specific field by key.

```go
traceField := zax.GetField(ctx, "trace_id")
fmt.Println(traceField.String) // "my-trace-id"
```

#### `GetSugared(ctx) []interface{}`
Returns fields as key-value pairs for `SugaredLogger`.

```go
sugar := logger.Sugar()
sugar.With(zax.GetSugared(ctx)...).Info("sugared log")
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

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| **Pure Zap** | ~35 | 112 | 1 |
| **Zax V2** | ~57 | 72 | 2 |
| Zax V1 | ~65 | 160 | 2 |

> 💡 **V2 uses 55% less memory** than V1 by storing only fields instead of the entire logger object.

<details>
<summary>📋 Full Benchmark Results</summary>


```


### benchmark
We have benchmarked Zax V2,V1 and Zap using the same fields. Here are the benchmark results:
As you can see in **V2** (Method with storing only fields in context, has better performance than V1 ( storing the whole logger object in context))

```
pkg: github.com/yuseferi/zax/v2
BenchmarkLoggingWithOnlyZap-10          103801226               35.56 ns/op          112 B/op          1 allocs/op
BenchmarkLoggingWithOnlyZap-10          98576570                35.56 ns/op          112 B/op          1 allocs/op
BenchmarkLoggingWithOnlyZap-10          100000000               35.24 ns/op          112 B/op          1 allocs/op
BenchmarkLoggingWithOnlyZap-10          100000000               34.85 ns/op          112 B/op          1 allocs/op
BenchmarkLoggingWithOnlyZap-10          100000000               34.98 ns/op          112 B/op          1 allocs/op
BenchmarkLoggingWithZaxV2-10            64324434                56.02 ns/op           72 B/op          2 allocs/op
BenchmarkLoggingWithZaxV2-10            63939517                56.98 ns/op           72 B/op          2 allocs/op
BenchmarkLoggingWithZaxV2-10            63374052                57.60 ns/op           72 B/op          2 allocs/op
BenchmarkLoggingWithZaxV2-10            63417358                57.37 ns/op           72 B/op          2 allocs/op
BenchmarkLoggingWithZaxV2-10            57964246                57.97 ns/op           72 B/op          2 allocs/op
BenchmarkLoggingWithZaxV1-10            54062712                66.40 ns/op          160 B/op          2 allocs/op
BenchmarkLoggingWithZaxV1-10            53155524                65.61 ns/op          160 B/op          2 allocs/op
BenchmarkLoggingWithZaxV1-10            54428521                64.19 ns/op          160 B/op          2 allocs/op
BenchmarkLoggingWithZaxV1-10            55420744                64.28 ns/op          160 B/op          2 allocs/op
BenchmarkLoggingWithZaxV1-10            55199061                64.50 ns/op          160 B/op          2 allocs/op
PASS
ok      github.com/yuseferi/zax/v2      56.919s
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
