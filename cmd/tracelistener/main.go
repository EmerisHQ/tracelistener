package main

import (
	"context"
	"flag"
	"syscall"
	"time"

	"github.com/allinbits/tracelistener/moduleprocessor"

	tracelistener2 "github.com/allinbits/tracelistener"
	blocktime2 "github.com/allinbits/tracelistener/blocktime"
	bulk2 "github.com/allinbits/tracelistener/bulk"
	config2 "github.com/allinbits/tracelistener/config"
	database2 "github.com/allinbits/tracelistener/database"

	"github.com/allinbits/tracelistener/utils/logging"

	"github.com/containerd/fifo"
	"go.uber.org/zap"
)

var Version = "not specified"

func main() {
	cfg, err := config2.Read()
	if err != nil {
		panic(err)
	}

	ca := readCLI()

	logger := buildLogger(cfg)

	logger.Infow("tracelistener", "version", Version)

	dpi, err := moduleprocessor.New(logger, cfg)
	if err != nil {
		logger.Fatal(err)
	}

	database2.RegisterMigration(dpi.DatabaseMigrations()...)
	database2.RegisterMigration(blocktime2.CreateTable)

	di, err := database2.New(cfg.DatabaseConnectionURL)
	if err != nil {
		logger.Fatal(err)
	}

	errChan := make(chan error)
	watcher := tracelistener2.TraceWatcher{
		WatchedOps: []tracelistener2.Operation{
			tracelistener2.WriteOp,
			tracelistener2.DeleteOp,
		},
		DataChan:       dpi.OpsChan(),
		ErrorChan:      errChan,
		Logger:         logger,
		DataSourcePath: cfg.FIFOPath,
	}

	if ca.existingDatabasePath != "" {
		importer := bulk2.Importer{
			Path:         ca.existingDatabasePath,
			TraceWatcher: watcher,
			Processor:    dpi,
			Logger:       logger,
			Database:     di,
		}

		if err := importer.Do(); err != nil {
			logger.Panicw("import error", "error", err)
		}

		return
	}

	if cfg.RunBlockWatcher {
		blw := blocktime2.New(
			di.Instance,
			cfg.ChainName,
			logger,
		)

		go connectTendermint(blw, logger)
	}

	ctx := context.Background()
	ff, err := fifo.OpenFifo(ctx, cfg.FIFOPath, syscall.O_CREAT|syscall.O_RDONLY|syscall.O_NONBLOCK, 0655)
	if err != nil {
		logger.Fatal(err)
	}

	if err := ff.Close(); err != nil {
		logger.Fatal(err)
	}

	go watcher.Watch()

	for {
		select {
		case e := <-errChan:
			logger.Errorw("watching error", "error", e)
		case e := <-dpi.ErrorsChan():
			te := e.(tracelistener2.TracingError)
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

func buildLogger(c *config2.Config) *zap.SugaredLogger {
	return logging.New(logging.LoggingConfig{
		LogPath: c.LogPath,
		Debug:   c.Debug,
	})
}

func connectTendermint(b *blocktime2.Watcher, l *zap.SugaredLogger) {
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

type cliArgs struct {
	existingDatabasePath string
}

func readCLI() cliArgs {
	ca := cliArgs{}

	flag.StringVar(&ca.existingDatabasePath, "import", "", "import LevelDB database data from the path given, usually you want to process `application.db'")
	flag.Parse()

	return ca
}
