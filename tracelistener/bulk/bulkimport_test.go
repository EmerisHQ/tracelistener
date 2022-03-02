package bulk_test

import (
	"testing"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	models "github.com/allinbits/demeris-backend-models/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener"
	bulk "github.com/allinbits/tracelistener/tracelistener/bulk"
	"github.com/allinbits/tracelistener/tracelistener/config"
	"github.com/allinbits/tracelistener/tracelistener/database"
	"github.com/allinbits/tracelistener/tracelistener/processor"
)

func TestImporterDo(t *testing.T) {
	var processorFunc tracelistener.DataProcessorFunc
	logger := zap.NewNop().Sugar()

	processorFunc = processor.New

	tests := []struct {
		name          string
		cfg           config.Config
		im            bulk.Importer
		connString    string
		expectedDBErr bool
		wantErr       bool
		startDB       bool
		checkInsert   bool
	}{
		{
			"Importer - no error",
			config.Config{
				FIFOPath:              "./tracelistener.fifo",
				DatabaseConnectionURL: "postgresql://demo:demo31621@/defaultdb?host=%2Ftmp%2Fdemo446173521&port=26257",
				ChainName:             "gaia",
				Debug:                 true,
			},
			bulk.Importer{
				Path: "/home/vitwit/go/src/github.com/allinbits/tracelistener/tracelistener/bulk/testdata/application.db",
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
			"",
			false,
			false,
			true,
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
			bulk.Importer{
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
					// tt.connString = ts.PGURL().String()
					tt.connString = "postgresql://demo:demo29922@/defaultdb?host=%2Ftmp%2Fdemo639490172&port=26257"
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

				if tt.checkInsert {
					// check auth rows
					var auth []models.AuthRow
					require.NoError(t,
						di.Instance.Exec(
							`select * from tracelistener.auth`,
							nil,
							&auth,
						),
					)
					require.NotZero(t, len(auth))
					require.NotNil(t, auth[0].Address)

					// check delegations
					var del []models.DelegationRow
					require.NoError(t,
						di.Instance.Exec(
							`select * from tracelistener.delegations`,
							nil,
							&del,
						),
					)
					require.NotZero(t, len(del))
					require.NotNil(t, del[0].Delegator)
					require.NotZero(t, del[0].Amount)
				}
			}

		})
	}
}
