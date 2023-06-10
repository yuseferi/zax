# Zax (zap with context)
[![codecov](https://codecov.io/github/yuseferi/zax/branch/codecov-integration/graph/badge.svg?token=64IHXT3ROF)](https://codecov.io/github/yuseferi/zax)
[![CodeQL](https://github.com/yuseferi/zax/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/yuseferi/zax/actions/workflows/github-code-scanning/codeql)
[![Check & Build](https://github.com/yuseferi/zax/actions/workflows/ci.yml/badge.svg)](https://github.com/yuseferi/zax/actions/workflows/ci.yml)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/yuseferi/zax)
[![Go Report Card](https://goreportcard.com/badge/github.com/yuseferi/zax)](https://goreportcard.com/report/github.com/yuseferi/zax)

Zax is a library that adds context to [Zap Logger](https://github.com/uber-go/zap) and makes it easier for Gophers to avoid generating logger boilerplates. By passing the logger as a parameter to functions, it enhances parameter functionality and avoids the need for multiple methods with explicit dependencies.

### Installation

```shell
  go get -u github.com/yuseferi/zax
```

### Usage:
To add something to the context and carry it along, simply use zap.Set:

    ctx = zax.Set(ctx, logger, []zap.Field{zap.String("trace_id", "my-trace-id")})

To retrieve a logger with the contexted fields, use zax.Get:

    zax.Get(ctx)

After that, you can use the output as a regular logger and perform logging operations:

```Go
zax.Get(ctx).Info(....)
zax.Get(ctx).Debug(....)
.....
```



##### example:
Let's say you want to generate a tracer at the entry point of your system and keep it until the process finishes:

```Go
func main() {
    logger, _ := zap.NewProduction()
    ctx := context.Background()
    s := NewServiceA(logger)
    ctx = zax.Set(ctx, logger, []zap.Field{zap.String("trace_id", "my-trace-id")})
    s.funcA(ctx)
}

type ServiceA struct {
logger *zap.Logger
}

func NewServiceA(logger *zap.Logger) *ServiceA {
    return &ServiceA{
        logger: logger,
    }
}

func (s *ServiceA) funcA(ctx context.Context) {
    s.logger.Info("func A") // it does not contain trace_id, you need to add it manually
    zax.Get(ctx).Info("func A") // it will logged with "trace_id" = "my-trace-id"
}

```
### benchmark
We have benchmarked Zax against Zap using the same fields. Here are the benchmark results:

```
BenchmarkLoggingWithOnlyZap-10          31756287                34.97 ns/op
BenchmarkLoggingWithOnlyZap-10          35056582                35.06 ns/op
BenchmarkLoggingWithOnlyZap-10          32982284                35.90 ns/op
BenchmarkLoggingWithOnlyZap-10          35061405                34.95 ns/op
BenchmarkLoggingWithOnlyZap-10          33266068                34.86 ns/op
BenchmarkLoggingWithZax-10              18442729                64.53 ns/op
BenchmarkLoggingWithZax-10              18592747                65.57 ns/op
BenchmarkLoggingWithZax-10              17492030                65.26 ns/op
BenchmarkLoggingWithZax-10              18640606                64.66 ns/op
BenchmarkLoggingWithZax-10              18700837                64.58 ns/op
```

### Contributing
We strongly believe in open-source ❤️😊. Please feel free to contribute by raising issues and submitting pull requests to make Zax even better!


Released under the [GNU GENERAL PUBLIC LICENSE](LICENSE).




