package main

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"syscall"
	"time"

	"github.com/containerd/fifo"
	"github.com/pkg/profile"
	"go.uber.org/zap"

	"github.com/emerishq/emeris-utils/logging"
	"github.com/emerishq/tracelistener/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener/blocktime"
	"github.com/emerishq/tracelistener/tracelistener/bulk"
	"github.com/emerishq/tracelistener/tracelistener/config"
	"github.com/emerishq/tracelistener/tracelistener/database"
	"github.com/emerishq/tracelistener/tracelistener/processor"
)

var (
	Version             = "not specified"
	SupportedSDKVersion = ""
)

func main() {
	if SupportedSDKVersion == "" {
		panic("missing sdk version at compile time, panic!")
	}

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

	if cfg.EnableCpuProfiling {
		logger.Debugw("enabling cpu profiling")
		defer profile.Start(profile.ProfilePath(".")).Stop()
	}

	logger.Infow("tracelistener", "version", Version, "supported_sdk_version", SupportedSDKVersion)

	dpi, err := processor.New(logger, cfg)
	if err != nil {
		logger.Fatal(err)
	}

	dpi.SetDBUpsertEnabled(true)

	dpi.StartBackgroundProcessing()

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
				wbUnits := p.SplitStatementToDBLimit()
				for _, wbUnit := range wbUnits {
					is := wbUnit.InterfaceSlice()
					if len(is) == 0 {
						continue
					}

					if err := di.Add(wbUnit.Statement, is); err != nil {
						logger.Errorw("database error",
							"error", err,
							"statement", wbUnit.Statement,
							"type", wbUnit.Type,
							"data", fmt.Sprint(wbUnit.Data),
						)
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
