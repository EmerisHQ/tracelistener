package logging

import (
	"os"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LoggingConfig struct {
	LogPath string
	Debug   bool
}

// New creates a zap.SugaredLogger configured following lc's directives.
// The non-debug logger returned by New is configured to output JSON-encoded structured logs to stdout.
// If lc.LogPath is empty, no log file is created, otherwise it will be compressed and rotated every 20MB, or when it reaches
// 28 days of usage. The last 3 copies are kept for backup.
func New(lc LoggingConfig) *zap.SugaredLogger {
	if lc.Debug {
		// we can safely ignore the error here
		dc, _ := zap.NewDevelopment()
		return dc.Sugar()
	}

	var cores []zapcore.Core

	if lc.LogPath != "" {
		l := &lumberjack.Logger{
			Filename:   lc.LogPath,
			MaxSize:    20,
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   true,
		}

		fileLogger := zapcore.AddSync(l)
		cores = append(cores, zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			fileLogger,
			zap.InfoLevel,
		))
	}

	stdWriter := zapcore.AddSync(os.Stdout)
	cores = append(cores, zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig()),
		stdWriter,
		zap.InfoLevel,
	))

	logger := zap.New(zapcore.NewTee(cores...))

	return logger.WithOptions(zap.AddCaller()).Sugar()
}
