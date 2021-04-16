package main

import (
	"context"
	"syscall"

	"github.com/allinbits/navigator-utils/logging"

	"github.com/allinbits/tracelistener"
	"github.com/allinbits/tracelistener/config"
	"github.com/allinbits/tracelistener/database"
	"github.com/allinbits/tracelistener/gaia_processor"
	"github.com/containerd/fifo"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		panic(err)
	}

	logger := buildLogger(cfg)

	var processorFunc tracelistener.DataProcessorFunc

	switch cfg.Type {
	case "gaia":
		processorFunc = gaia_processor.New
	default:
		logger.Panicw("no processor associated with type", "type", cfg.Type)
	}

	dpi, err := processorFunc(logger, cfg)
	if err != nil {
		logger.Fatal(err)
	}

	database.RegisterMigration(dpi.DatabaseMigrations...)

	di, err := database.New(cfg.DatabaseConnectionURL)
	if err != nil {
		logger.Fatal(err)
	}

	ctx := context.Background()
	f, err := fifo.OpenFifo(ctx, cfg.FIFOPath, syscall.O_CREAT|syscall.O_RDONLY, 0655)
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
			for _, p := range b {
				for _, asd := range p.Data {
					logger.Debugw("writeback unit", "data", asd)
				}

				is := p.InterfaceSlice()
				if len(is) == 0 {
					continue
				}

				if err := di.Add(p.DatabaseExec, is); err != nil {
					logger.Error("database error ", err)
				}
			}
		}
	}
}

func buildLogger(c *config.Config) *zap.SugaredLogger {
	return logging.New(logging.LoggingConfig{
		LogPath: c.LogPath,
		Debug:   c.Debug,
	})
}
