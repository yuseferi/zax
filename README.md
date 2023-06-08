# Zax (zap with context)
[![codecov](https://codecov.io/github/yuseferi/zax/branch/codecov-integration/graph/badge.svg?token=64IHXT3ROF)](https://codecov.io/github/yuseferi/zax)
[![CodeQL](https://github.com/yuseferi/zax/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/yuseferi/zax/actions/workflows/github-code-scanning/codeql)
[![Check & Build](https://github.com/yuseferi/zax/actions/workflows/ci.yml/badge.svg)](https://github.com/yuseferi/zax/actions/workflows/ci.yml)

Basically this adds context to [Zap Logger](https://github.com/uber-go/zap), and make it easier to for the Gophers to do not generate logger boiler plates.
Passing logger as a parameter to function increase parameters functionalities and worse than couple lots of methods with a explicit dependency.


### Installation

```shell
  go get -u github.com/yuseferi/zax
```

### Usage:
when you add something to context and would like to carry with context , you just need to add it to context with calling `zap.Set`

    ctx = zax.Set(ctx, logger, []zap.Field{zap.String("trace_id", "my-trace-id")})

and when you want to log context fields, just use    
        
    zax.Get(ctx)



##### example:
you want to generate a tracer on entry point of your system and want to keep it until the process finished.

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

### Contributing
I strongly believe in open-source :), feel free to make it better with raising issues and PRs.


Released under the [GNU GENERAL PUBLIC LICENSE](LICENSE).




