package log

import (
	"context"
	"os"
	"sync"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	JsonEncoderName    = "json"
	ConsoleEncoderName = "console"
)

type LevelString string

const (
	LevelDebug  LevelString = "debug"
	LevelInfo               = "info"
	LevelWarn               = "warn"
	LevelError              = "error"
	LevelDPanic             = "dpanic"
	LevelPanic              = "panic"
	LevelFatal              = "fatal"
)

func (l LevelString) ToZapLevel() (zapcore.Level, error) {
	return zapcore.ParseLevel(l.String())
}

func (l LevelString) ToZapLevelOrDefault(def zapcore.Level) zapcore.Level {
	lvl, err := l.ToZapLevel()
	if err != nil {
		return def
	}
	return lvl
}

func (l LevelString) String() string {
	return string(l)
}

var DefaultLoggerConfig = &Config{
	Debug:   false,
	Encoder: JsonEncoderName,
}

var DefaultEncoderConfig = zapcore.EncoderConfig{
	TimeKey:        "ts",
	LevelKey:       "level",
	NameKey:        "logger",
	CallerKey:      "caller",
	FunctionKey:    zapcore.OmitKey,
	MessageKey:     "msg",
	StacktraceKey:  "stacktrace",
	LineEnding:     zapcore.DefaultLineEnding,
	EncodeLevel:    zapcore.LowercaseLevelEncoder,
	EncodeTime:     zapcore.RFC3339TimeEncoder,
	EncodeDuration: zapcore.SecondsDurationEncoder,
	EncodeCaller:   zapcore.ShortCallerEncoder,
}

var LogEncoderMap = map[string]zapcore.Encoder{
	JsonEncoderName:    zapcore.NewJSONEncoder(DefaultEncoderConfig),
	ConsoleEncoderName: zapcore.NewConsoleEncoder(DefaultEncoderConfig),
}

// Re-export zap field constructors
var (
	Any        = zap.Any
	Bool       = zap.Bool
	Duration   = zap.Duration
	Float64    = zap.Float64
	Int        = zap.Int
	Int64      = zap.Int64
	Skip       = zap.Skip
	String     = zap.String
	Strings    = zap.Strings
	Stringer   = zap.Stringer
	Time       = zap.Time
	Uint       = zap.Uint
	Uint32     = zap.Uint32
	Uint64     = zap.Uint64
	Uintptr    = zap.Uintptr
	ByteString = zap.ByteString
	Error      = zap.Error
	Reflect    = zap.Reflect
)

var (
	initOnce sync.Once
)

type CtxFunc func(ctx context.Context) []zap.Field

func For(ctx context.Context, ctxFuncs ...CtxFunc) *zap.Logger {
	logger := GetContextLogData(ctx)
	if logger != nil {
		var fields []zap.Field
		for _, fn := range ctxFuncs {
			fields = append(fields, fn(ctx)...)
		}
		return logger.With(fields...)
	}

	logger = Bg()
	var fields []zap.Field
	for _, fn := range ctxFuncs {
		fields = append(fields, fn(ctx)...)
	}
	logger = logger.With(fields...)

	if span := trace.SpanFromContext(ctx); span != nil {
		spanCtx := span.SpanContext()
		fields := []zapcore.Field{
			zap.String("trace_id", spanCtx.TraceID().String()),
			zap.String("span_id", spanCtx.SpanID().String()),
		}

		for _, ctxField := range ctxFuncs {
			fields = append(fields, ctxField(ctx)...)
		}
	}

	return logger
}

func Bg() *zap.Logger {
	initOnce.Do(func() {
		logger, _ := newLogger(DefaultLoggerConfig)
		zap.ReplaceGlobals(logger)
	})
	return zap.L()
}

func InitLogger(cfg *Config) (func(), error) {
	var (
		logger *zap.Logger
		err    error
	)

	initOnce.Do(func() {
		logger, err = newLogger(cfg)
		if err != nil {
			return
		}
		zap.ReplaceGlobals(logger)
	})

	return func() {
		_ = logger.Sync()
	}, err
}

func newLogger(cfg *Config) (*zap.Logger, error) {
	if err := validateLogConfig(cfg); err != nil {
		return nil, err
	}

	encoder := getEncoder(cfg.Encoder)
	minLevel := zapcore.InfoLevel
	if cfg.Debug {
		minLevel = zapcore.DebugLevel
	}

	cores := make([]zapcore.Core, 0, len(cfg.Sampler)+1)
	coveredLevels := make(map[zapcore.Level]struct{})
	for _, samplerCfg := range cfg.Sampler {
		samplerCore := newSamplerCode(encoder, samplerCfg, minLevel)
		cores = append(cores, samplerCore)

		fromLevel := samplerCfg.LevelRange.From.ToZapLevelOrDefault(minLevel)
		toLevel := samplerCfg.LevelRange.To.ToZapLevelOrDefault(zapcore.FatalLevel)
		for lvl := fromLevel; lvl <= toLevel; lvl++ {
			coveredLevels[lvl] = struct{}{}
		}
	}
	defaultCore := zapcore.NewCore(encoder, os.Stderr, zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		if lvl < minLevel {
			return false
		}
		if _, covered := coveredLevels[lvl]; covered {
			return false
		}
		return true
	}))
	cores = append(cores, defaultCore)

	core := zapcore.NewTee(cores...)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.PanicLevel))

	return logger, nil
}

func validateLogConfig(cfg *Config) error {
	if cfg == nil {
		return nil
	}

	if len(cfg.Sampler) > 0 {
		// make sure level ranges does not overlap
		levelRanges := make([]zapcore.Level, 0, len(cfg.Sampler)*2)
		for _, sc := range cfg.Sampler {
			fromLevel := sc.LevelRange.From.ToZapLevelOrDefault(zapcore.DebugLevel)
			toLevel := sc.LevelRange.To.ToZapLevelOrDefault(zapcore.FatalLevel)
			if fromLevel > toLevel {
				return ErrSamplerLevelRangeInvalid
			}
			levelRanges = append(levelRanges, fromLevel, toLevel)
		}
		for i := 0; i < len(levelRanges); i += 2 {
			for j := i + 2; j < len(levelRanges); j += 2 {
				if levelRanges[i] <= levelRanges[j+1] && levelRanges[j] <= levelRanges[i+1] {
					return ErrSamplerLevelRangeOverlap
				}
			}
		}
	}

	return nil
}

func getEncoder(encoderName string) zapcore.Encoder {
	encoder, ok := LogEncoderMap[encoderName]
	if !ok {
		encoder = LogEncoderMap[JsonEncoderName]
	}
	zap.L()
	return encoder
}

func newSamplerCode(encoder zapcore.Encoder, samplerCfg SamplerConfig, minLevel zapcore.Level) zapcore.Core {
	fromLevel := samplerCfg.LevelRange.From.ToZapLevelOrDefault(minLevel)
	toLevel := samplerCfg.LevelRange.To.ToZapLevelOrDefault(zapcore.FatalLevel)
	return zapcore.NewSamplerWithOptions(
		zapcore.NewCore(encoder, os.Stderr, zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= fromLevel && lvl <= toLevel
		})),
		samplerCfg.Interval,
		samplerCfg.First,
		samplerCfg.Thereafter,
	)
}
