package main

import (
	"context"
	"os"
	"syscall"

	"github.com/allinbits/tracelistener/database"

	"github.com/allinbits/tracelistener/gaia_processor"

	"github.com/allinbits/tracelistener"

	"go.uber.org/zap/zapcore"

	"go.uber.org/zap"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/containerd/fifo"
)

func main() {
	config, err := readConfig()
	if err != nil {
		panic(err)
	}

	logger := logging(config)

	dpi, err := gaia_processor.New(logger)
	if err != nil {
		logger.Fatal(err)
	}

	database.RegisterMigration(dpi.DatabaseMigrations...)

	di, err := database.New(config.DatabaseConnectionURL)
	if err != nil {
		logger.Fatal(err)
	}

	ctx := context.Background()
	f, err := fifo.OpenFifo(ctx, config.FIFOPath, syscall.O_CREAT|syscall.O_RDONLY, 0655)
	if err != nil {
		logger.Fatal(err)
	}

	errChan := make(chan error)
	watcher := tracelistener.TraceWatcher{
		DataSource: f,
		WatchedOps: []tracelistener.Operation{
			tracelistener.WriteOp,
			tracelistener.DeleteOp,
		},
		DataChan:  dpi.OpsChan,
		ErrorChan: errChan,
		Logger:    logger,
	}

	go watcher.Watch()

	for {
		select {
		case e := <-errChan:
			logger.Error("watching error", e)
		case b := <-dpi.WritebackChan:
			logger.Debug("writeback packet", b)
			for _, p := range b {
				if err := di.Add(p.DatabaseExec, p.Data); err != nil {
					logger.Error("database insert error ", err)
				}
			}
		}
	}
}

func logging(c *Config) *zap.SugaredLogger {
	if c.Debug {
		// we can safely ignore the error here
		dc, _ := zap.NewDevelopment()
		return dc.Sugar()
	}

	var cores []zapcore.Core

	l := &lumberjack.Logger{
		Filename:   c.LogPath,
		MaxSize:    20,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	}

	fileLogger := zapcore.AddSync(l)
	jsonWriter := zapcore.AddSync(os.Stdout)

	cores = append(cores, zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		fileLogger,
		zap.InfoLevel,
	))

	// we use development encoder config in CLI output because it's easier to read
	cores = append(cores, zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		jsonWriter,
		zap.InfoLevel,
	))

	logger := zap.New(zapcore.NewTee(cores...))

	return logger.WithOptions(zap.AddCaller()).Sugar()
}
