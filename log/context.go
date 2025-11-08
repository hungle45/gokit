package log

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/status"
)

type contextLogKey struct{}

var (
	contextLog = contextLogKey{}
)

func NewLoggerCtx(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, contextLog, logger)
}

func GetContextLogData(ctx context.Context) *zap.Logger {
	logger, ok := ctx.Value(contextLog).(*zap.Logger)
	if !ok {
		return nil
	}
	return logger
}

func GRPCError(err error) zap.Field {
	return zap.Reflect("error_details", status.Convert(err).Details())
}
