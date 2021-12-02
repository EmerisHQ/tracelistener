package bulk

import (
	"testing"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/allinbits/tracelistener/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener/config"
	"github.com/allinbits/tracelistener/tracelistener/database"
	"github.com/allinbits/tracelistener/tracelistener/gaia_processor"
)

func TestImporterDo(t *testing.T) {
	var processorFunc tracelistener.DataProcessorFunc
	logger := zap.NewNop().Sugar()

	processorFunc = gaia_processor.New

	tests := []struct {
		name          string
		cfg           config.Config
		im            Importer
		connString    string
		expectedDBErr bool
		wantErr       bool
		startDB       bool
	}{
		{
			"Importer - no error",
			config.Config{
				FIFOPath:              "./tracelistener.fifo",
				DatabaseConnectionURL: "postgres://demo:demo32622@127.0.0.1:26257?sslmode=require",
				ChainName:             "gaia",
				Debug:                 true,
			},
			Importer{
				Path: "./testdata/application.db",
				TraceWatcher: tracelistener.TraceWatcher{
					DataSourcePath: "./tracelistener.fifo",
					WatchedOps: []tracelistener.Operation{
						tracelistener.WriteOp,
						tracelistener.DeleteOp,
					},
					ErrorChan: make(chan error),
					Logger:    zap.NewNop().Sugar(),
				},
				Logger: zap.NewNop().Sugar(),
			},
			"",
			false,
			false,
			true,
		},
		{
			"cannot open chain database - error",
			config.Config{
				FIFOPath:              "./tracelistener.fifo",
				DatabaseConnectionURL: "postgres://demo:demo32622@127.0.0.1:26257?sslmode=require",
				ChainName:             "gaia",
				Debug:                 true,
			},
			Importer{
				Path: "./application.db",
				TraceWatcher: tracelistener.TraceWatcher{
					DataSourcePath: "/home/vitwit/go/src/github.com/allinbits/tracelistener/tracelistener.fifo",
					WatchedOps: []tracelistener.Operation{
						tracelistener.WriteOp,
						tracelistener.DeleteOp,
					},
					ErrorChan: make(chan error),
					Logger:    zap.NewNop().Sugar(),
				},
				Logger: zap.NewNop().Sugar(),
			},
			"invalid connection",
			true,
			true,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			dpi, err := processorFunc(logger, &tt.cfg)
			require.NoError(t, err)

			tt.im.TraceWatcher.DataChan = dpi.OpsChan()
			tt.im.Processor = dpi

			if tt.startDB {
				ts, err := testserver.NewTestServer()
				require.NoError(t, err)
				require.NoError(t, ts.WaitForInit())
				defer func() {
					ts.Stop()
				}()

				if tt.connString == "" {
					tt.connString = ts.PGURL().String()
				}
			}

			di, err := database.New(tt.connString)
			if tt.expectedDBErr {
				require.Error(t, err)
				require.Nil(t, di)
				return
			}

			tt.im.Database = di

			err = tt.im.Do()
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
