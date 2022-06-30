package logging

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/zap"
)

type ctxKey string

const (
	LoggerKey = "logger"

	CorrelationIDName         ctxKey = "correlation_id"
	IntCorrelationIDName      ctxKey = "int_correlation_id"
	ExternalCorrelationIDName string = "X-Correlation-Id"
)

// AddLoggerMiddleware adds a logger to the gin context, with some fields
// populated (correlation ID, requests params, ...).
// The logger can be retrieved by calling GetLoggerFromContext(c).
func AddLoggerMiddleware(l *zap.SugaredLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		addLogger(c, l)
	}
}

func addLogger(c *gin.Context, l *zap.SugaredLogger) {
	ctx := c.Request.Context()

	correlationID := c.Request.Header.Get(ExternalCorrelationIDName)

	if correlationID != "" {
		ctx = context.WithValue(ctx, CorrelationIDName, correlationID)
		c.Writer.Header().Set(ExternalCorrelationIDName, correlationID)
		l = l.With(string(CorrelationIDName), correlationID)
	}

	id, err := uuid.NewV4()
	if err != nil {
		l.Errorf("Error while creating new internal correlation id error: %w", err)
	}

	ctx = context.WithValue(ctx, IntCorrelationIDName, id.String())
	l = l.With(string(IntCorrelationIDName), id)

	for _, p := range c.Params {
		l = l.With(p.Key, p.Value)
	}

	if len(c.Request.URL.RawQuery) > 0 {
		l = l.With("query", c.Request.URL.RawQuery)
	}

	c.Set(LoggerKey, l)

	c.Request = c.Request.WithContext(ctx)

	c.Next()
}

// AddCorrelationIDToLogger takes correlation ID from the request context and
// enriches the logger with them. The param logger cannot be nil.
func AddCorrelationIDToLogger(c *gin.Context, l *zap.SugaredLogger) *zap.SugaredLogger {
	if c == nil {
		return l
	}

	// note: correlation IDs are in the request context, not in the gin context
	ctx := c.Request.Context()

	return l.With(
		string(CorrelationIDName), ctx.Value(CorrelationIDName),
		string(IntCorrelationIDName), ctx.Value(IntCorrelationIDName),
	)
}
