# log

Lightweight logging helpers and opinionated zap logger configuration used across the project.

This package wraps and configures go.uber.org/zap with:

- a simple `Config` struct (JSON/YAML friendly via mapstructure)
- encoder selection (json or console)
- optional level-based sampling rules
- helpers to store/retrieve a logger from a context and to enrich logs with context-derived fields

## Key API

- `type Config` — top-level configuration struct (fields below).
- `InitLogger(cfg *Config) (func(), error)` — initialize the global logger. Returns a shutdown `func()` you can call to `Sync()` the logger, and any error encountered.
- `For(ctx context.Context, ctxFuncs ...CtxFunc) *zap.Logger` — get a logger for a request context. If a logger has been stored in the context (via `NewLoggerCtx`) it will be used, otherwise the package global logger is returned.
- `Bg() *zap.Logger` — returns the package/global logger (creates a default logger on first call).
- `NewLoggerCtx(ctx, logger)` / `GetContextLogData(ctx)` — helpers for putting/getting a `*zap.Logger` in a `context.Context`.
- `GRPCError(err error) zap.Field` — helper to attach gRPC error details to a log entry.
- `CtxFunc` — type for functions that extract zap fields from a `context.Context` and attach them when calling `For`.

Errors:

- `ErrSamplerLevelRangeInvalid` — sampler from/to inverted.
- `ErrSamplerLevelRangeOverlap` — two sampler ranges overlap.

Constants and encoder names:

- `JsonEncoderName = "json"`
- `ConsoleEncoderName = "console"`

## Configuration

The `Config` struct has these fields:

- `Debug` (bool) — enables debug-level logging when true.
- `Encoder` (string) — encoder name: `json` or `console`.
- `Sampler` ([]SamplerConfig) — optional list of sampler rules. Each `SamplerConfig` contains:
  - `LevelRange` (`from`, `to`) — level range for the sampler (values like `debug`, `info`, `warn`, `error`, ...)
  - `Interval` — a duration (e.g. `500ms`) used by the zap sampler
  - `First`, `Thereafter` — sampler parameters (how many events to allow, then thereafter)

Sampler ranges must not overlap and `from` must be <= `to`. The package will return `ErrSamplerLevelRangeInvalid` or `ErrSamplerLevelRangeOverlap` when validation fails.

Example YAML:

```yaml
debug: true
encoder: json
sampler:
  - level_range:
      from: debug
      to: info
    interval: 500ms
    first: 10
    thereafter: 50
```

## Usage

A minimal usage example:

```go
var cfg *log.Config
// load cfg from file/env using viper (see `main.go` in the repo for a full example)

closer, err := log.InitLogger(cfg)
if err != nil {
    panic(err)
}
defer closer()

// get a logger for a context and add structured fields
logger := log.Bg().With(zap.Int32("abc", 123))
logger.Info("hello world")
```

If you want to use a custom `*zap.Logger` for a specific context (for example, a request-scoped logger), store it with `NewLoggerCtx`:

```go
ctx = log.NewLoggerCtx(ctx, myZapLogger)
logger := log.For(ctx)
```

`For` also accepts `CtxFunc` functions which extract zap fields from a context and automatically attach them to the returned logger.

## Notes

- The package provides sensible defaults when no config is provided (see `DefaultLoggerConfig`). Calling `Bg()` or `For` will initialize a default logger the first time.
- Encoder selection falls back to `json` if an unknown encoder name is provided.
- Remember to call the `func()` returned by `InitLogger` (typically deferred in `main`) to flush buffered logs.

## Troubleshooting

- If InitLogger returns `ErrSamplerLevelRangeInvalid` or `ErrSamplerLevelRangeOverlap`, check your sampler `level_range` entries in the config.
