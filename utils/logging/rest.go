package logging

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LogRequest is a gin middleware that logs useful informations on each request as they come.
func LogRequest(l *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
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
}
