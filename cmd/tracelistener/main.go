package main

import (
	"context"
	"syscall"
	"time"

	"github.com/allinbits/demeris-backend/tracelistener/blocktime"

	"github.com/allinbits/demeris-backend/utils/logging"

	"github.com/allinbits/demeris-backend/tracelistener"
	"github.com/allinbits/demeris-backend/tracelistener/config"
	"github.com/allinbits/demeris-backend/tracelistener/database"
	"github.com/allinbits/demeris-backend/tracelistener/gaia_processor"
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

	database.RegisterMigration(dpi.DatabaseMigrations()...)
	database.RegisterMigration(blocktime.CreateTable)

	di, err := database.New(cfg.DatabaseConnectionURL)
	if err != nil {
		logger.Fatal(err)
	}

	blw := blocktime.New(
		di.Instance,
		cfg.ChainName,
		logger,
	)

	go connectTendermint(blw, logger)

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
		DataChan:  dpi.OpsChan(),
		ErrorChan: errChan,
		Logger:    logger,
	}

	go watcher.Watch()

	for {
		select {
		case e := <-errChan:
			logger.Errorw("watching error", "error", e)
		case e := <-dpi.ErrorsChan():
			te := e.(tracelistener.TracingError)
			logger.Errorw(
				"error while processing data",
				"error", te.InnerError,
				"data", te.Data,
				"moduleName", te.Module)
		case b := <-dpi.WritebackChan():
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

func connectTendermint(b *blocktime.Watcher, l *zap.SugaredLogger) {
	connected := false

	for !connected {
		if err := b.Connect(); err != nil {
			l.Errorw("cannot connect to tendermint rpc, retrying in 5 seconds", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}

		connected = true
	}
}
