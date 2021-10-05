package logging

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var l *zap.Logger

// LogRequest is a gin middleware that logs useful informations on each request as they come.
func LogRequest(logger *zap.Logger) gin.HandlerFunc {
	l = logger
	return log
}

func log(c *gin.Context) {
	start := time.Now()
	// some evil middlewares modify this values
	path := c.Request.URL.Path
	query := c.Request.URL.RawQuery

	l.Info(path,
		zap.Int("status", c.Writer.Status()),
		zap.String("method", c.Request.Method),
		zap.String("path", path),
		zap.String("query", query),
		zap.String("ip", c.ClientIP()),
		zap.String("user-agent", c.Request.UserAgent()),
		zap.String("time", start.Format(time.RFC3339)),
	)

	c.Next()
}
