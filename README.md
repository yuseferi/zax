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

To retrieve an specific field in the context you can use zax.GetField:

     zax.GetField(ctx, "trace_id")

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

### Contributing
We strongly believe in open-source ❤️😊. Please feel free to contribute by raising issues and submitting pull requests to make Zax even better!


Released under the [GNU GENERAL PUBLIC LICENSE](LICENSE).




