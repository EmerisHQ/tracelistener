package main

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"syscall"
	"time"

	"github.com/allinbits/tracelistener/tracelistener/bulk"

	"github.com/allinbits/tracelistener/tracelistener/blocktime"

	"github.com/allinbits/emeris-utils/logging"

	"github.com/allinbits/tracelistener/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener/config"
	"github.com/allinbits/tracelistener/tracelistener/database"
	"github.com/allinbits/tracelistener/tracelistener/gaia_processor"
	"github.com/containerd/fifo"
	"go.uber.org/zap"
)

var Version = "not specified"

func main() {
	ca := readCLI()

	if ca.bulkImportSupportedModules {
		fmt.Println("Import-able modules list:", strings.Join(bulk.ImportableModulesList(), ", "))
		return
	}

	cfg, err := config.Read()
	if err != nil {
		panic(err)
	}

	logger := buildLogger(cfg)

	logger.Infow("tracelistener", "version", Version)

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

	errChan := make(chan error)
	watcher := tracelistener.TraceWatcher{
		WatchedOps: []tracelistener.Operation{
			tracelistener.WriteOp,
			tracelistener.DeleteOp,
		},
		DataChan:       dpi.OpsChan(),
		ErrorChan:      errChan,
		Logger:         logger,
		DataSourcePath: cfg.FIFOPath,
	}

	if ca.existingDatabasePath != "" {
		importer := bulk.Importer{
			Path:         ca.existingDatabasePath,
			TraceWatcher: watcher,
			Processor:    dpi,
			Logger:       logger,
			Database:     di,
			Modules:      ca.bulkImportModulesSlice(),
		}

		if err := importer.Do(); err != nil {
			logger.Panicw("import error", "error", err)
		}

		return
	}

	blw := blocktime.New(
		di.Instance,
		cfg.ChainName,
		logger,
	)

	go connectTendermint(blw, logger)

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

				wbUnits := p.SplitStatementToDBLimit()
				for _, wbUnit := range wbUnits {
					is := wbUnit.InterfaceSlice()
					if len(is) == 0 {
						continue
					}

					if err := di.Add(wbUnit.DatabaseExec, is); err != nil {
						logger.Error("database error ", err)
					}
				}
			}
		}
	}
}

func buildLogger(c *config.Config) *zap.SugaredLogger {
	return logging.New(logging.LoggingConfig{
		LogPath: c.LogPath,
		Debug:   c.Debug,
		JSON:    c.JSONLogs,
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

type cliArgs struct {
	existingDatabasePath       string
	bulkImportModules          string
	bulkImportSupportedModules bool
}

func (c cliArgs) bulkImportModulesSlice() []string {
	if c.bulkImportModules == "" {
		return nil
	}

	s := strings.Split(c.bulkImportModules, ",")
	for i := 0; i < len(s); i++ {
		s[i] = strings.TrimSpace(s[i])
	}

	return s
}

func readCLI() cliArgs {
	ca := cliArgs{}

	flag.StringVar(&ca.existingDatabasePath, "import", "", "import LevelDB database data from the path given, usually you want to process `application.db'; will import all modules listed by `-import-modules-list` if `-import-modules` is not specified")
	flag.StringVar(&ca.bulkImportModules, "import-modules", "", "comma-separated list of modules to be imported")
	flag.BoolVar(&ca.bulkImportSupportedModules, "import-modules-list", false, "list supported modules in bulk import mode")
	flag.Parse()

	return ca
}
