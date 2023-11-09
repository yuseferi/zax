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
  go get -u github.com/yuseferi/zax/v2
```

### Usage:
To add something to the context and carry it along, simply use zap.Set:

    ctx = zax.Set(ctx, []zap.Field{zap.String("trace_id", "my-trace-id")})

To retrieve stored zap fields in context, use zax.Get:

     zax.Get(ctx)  // this retrive stored zap fields in context 

To retrieve stored zap fields in context and log them :

     logger.With(zax.Get(ctx)...).Info("just a test record")


After that, you can use the output as a regular logger and perform logging operations:

```Go
logger.With(zax.Get(ctx)...).Info("message")
logger.With(zax.Get(ctx)...).Debug("message")
```



##### example:
Let's say you want to generate a tracer at the entry point of your system and keep it until the process finishes:

```Go
func main() {
    logger, _ := zap.NewProduction()
    ctx := context.Background()
    s := NewServiceA(logger)
    ctx = zax.Set(ctx, zap.String("trace_id", "my-trace-id"))  
    // and if you want to add multiple of them at once
    //ctx = zax.Set(ctx, []zap.Field{zap.String("trace_id", "my-trace-id"),zap.String("span_id", "my-span-id")})
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
	s.logger.With(zax.Get(ctx)...).Info("func A") // it will logged with "trace_id" = "my-trace-id"
}

```
### benchmark
We have benchmarked Zax against Zap using the same fields. Here are the benchmark results:

```
pkg: github.com/yuseferi/zax/v2
BenchmarkLoggingWithOnlyZap-10          344718321               35.23 ns/op
BenchmarkLoggingWithOnlyZap-10          340526908               36.74 ns/op
BenchmarkLoggingWithOnlyZap-10          337279976               36.17 ns/op
BenchmarkLoggingWithOnlyZap-10          338681052               36.18 ns/op
BenchmarkLoggingWithOnlyZap-10          339414484               35.48 ns/op
BenchmarkLoggingWithZax-10              201602071               56.58 ns/op
BenchmarkLoggingWithZax-10              213688218               57.44 ns/op
BenchmarkLoggingWithZax-10              206059045               56.66 ns/op
BenchmarkLoggingWithZax-10              211847756               58.14 ns/op
BenchmarkLoggingWithZax-10              210184916               56.69 ns/op

```

### Contributing
We strongly believe in open-source ❤️😊. Please feel free to contribute by raising issues and submitting pull requests to make Zax even better!


Released under the [GNU GENERAL PUBLIC LICENSE](LICENSE).




