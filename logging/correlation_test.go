package logging

import (
	"context"
	"go.uber.org/zap/zaptest/observer"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func makeTestContext() *gin.Context {
	c, _ := gin.CreateTestContext(nil)
	ctx := context.WithValue(context.Background(), IntCorrelationIDName, "anything")
	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, "http://something", nil)
	c.Request = request
	return c
}

func Test_AddCorrelationIDToLogger_Nil_Context(t *testing.T) {
	assert := assert.New(t)

	base := zap.NewExample().Sugar()

	assert.NotPanics(func() {
		logger := AddCorrelationIDToLogger(nil, base)
		assert.Equal(base, logger)
	})
}

func Test_AddCorrelationIDToLogger_Nil_Logger(t *testing.T) {
	assert := assert.New(t)

	c := makeTestContext()

	assert.Panics(func() {
		AddCorrelationIDToLogger(c, nil)
	})
}

func Test_AddCorrelationIDToLogger(t *testing.T) {
	assert := assert.New(t)

	c := makeTestContext()

	observedZapCore, observedLogs := observer.New(zap.InfoLevel)
	observedLogger := zap.New(observedZapCore)

	logger := AddCorrelationIDToLogger(c, observedLogger.Sugar())
	logger.Info("test")

	assert.Equal("anything", observedLogs.All()[0].ContextMap()[string(IntCorrelationIDName)])
}
