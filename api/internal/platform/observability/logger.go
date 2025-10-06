package observability

import (
	"context"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/hanko-field/api/internal/platform/requestctx"
)

const defaultLogLevel = "info"

// NewLogger constructs a production-ready zap logger emitting structured JSON.
func NewLogger() (*zap.Logger, error) {
	level := zap.NewAtomicLevel()
	if err := level.UnmarshalText([]byte(strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL"))))); err != nil {
		// Fallback to default level when env var is unset or invalid.
		_ = level.UnmarshalText([]byte(defaultLogLevel))
	}

	encoderCfg := zapcore.EncoderConfig{
		MessageKey: "message",
		TimeKey:    "timestamp",
		LevelKey:   "severity",
		EncodeTime: zapcore.RFC3339NanoTimeEncoder,
		EncodeLevel: func(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(strings.ToUpper(level.String()))
		},
		CallerKey:     "caller",
		StacktraceKey: "stacktrace",
	}

	cfg := zap.Config{
		Level:             level,
		Encoding:          "json",
		EncoderConfig:     encoderCfg,
		OutputPaths:       []string{"stdout"},
		ErrorOutputPaths:  []string{"stderr"},
		DisableCaller:     false,
		DisableStacktrace: true,
	}

	return cfg.Build()
}

// WithLogger injects the logger into the provided context.
func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return requestctx.WithLogger(ctx, logger)
}

// FromContext retrieves the logger from context, defaulting to a no-op logger.
func FromContext(ctx context.Context) *zap.Logger {
	return requestctx.Logger(ctx)
}

// PrintfAdapter adapts zap to printf-style logging interfaces.
type PrintfAdapter struct {
	logger *zap.SugaredLogger
}

// NewPrintfAdapter creates a PrintfAdapter backed by the supplied logger.
func NewPrintfAdapter(logger *zap.Logger) PrintfAdapter {
	if logger == nil {
		logger = zap.NewNop()
	}
	return PrintfAdapter{logger: logger.Sugar()}
}

// Printf implements the Printf-style logging expected by legacy interfaces.
func (a PrintfAdapter) Printf(format string, args ...any) {
	a.logger.Infof(format, args...)
}

// WithRequestFields augments the logger with standard request-scoped fields.
func WithRequestFields(logger *zap.Logger, fields ...zap.Field) *zap.Logger {
	if logger == nil {
		logger = zap.NewNop()
	}
	return logger.With(fields...)
}
