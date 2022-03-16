package bulk_test

import (
	"testing"
	"time"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	models "github.com/allinbits/demeris-backend-models/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener"

	"github.com/allinbits/tracelistener/tracelistener/blocktime"
	bulk "github.com/allinbits/tracelistener/tracelistener/bulk"
	"github.com/allinbits/tracelistener/tracelistener/config"
	"github.com/allinbits/tracelistener/tracelistener/database"
	"github.com/allinbits/tracelistener/tracelistener/processor"
)

func TestImporterDo(t *testing.T) {
	var processorFunc tracelistener.DataProcessorFunc
	loggerRaw, _ := zap.NewProduction()
	logger := loggerRaw.Sugar()

	processorFunc = processor.New

	expectedBalances := []string{
		"4fea76427b8345861e80a3540a8a9d936fd39391",
		"93354845030274cd4bf1686abd60ab28ec52e1a7",
		"28830cb550d76d286c72c1d91782fdca52cbd539",
	}

	expectedDelegations := []string{
		"28830cb550d76d286c72c1d91782fdca52cbd539",
	}

	expectedValidators := []string{
		"cosmosvaloper19zpsed2s6akjsmrjc8v30qhaeffvh4fec7lfcg",
	}

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
				FIFOPath:  "./tracelistener.fifo",
				ChainName: "gaia",
				Debug:     true,
			},
			bulk.Importer{
				Path: "./testdata/application.db",
				TraceWatcher: tracelistener.TraceWatcher{
					DataSourcePath: "./tracelistener.fifo",
					WatchedOps: []tracelistener.Operation{
						tracelistener.WriteOp,
						tracelistener.DeleteOp,
					},
					ErrorChan: make(chan error),
					Logger:    logger,
				},
				Logger: logger,
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
				FIFOPath:  "./tracelistener.fifo",
				ChainName: "gaia",
				Debug:     true,
			},
			bulk.Importer{
				Path: "./application.db",
				TraceWatcher: tracelistener.TraceWatcher{
					DataSourcePath: "./tracelistener.fifo",
					WatchedOps: []tracelistener.Operation{
						tracelistener.WriteOp,
						tracelistener.DeleteOp,
					},
					ErrorChan: make(chan error),
					Logger:    logger,
				},
				Logger: logger,
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
					tt.connString = ts.PGURL().String()
				}

				database.RegisterMigration(dpi.DatabaseMigrations()...)
				database.RegisterMigration(blocktime.CreateTable)

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

				time.Sleep(2 * time.Second)

				if tt.checkInsert {
					// we are expecting data to be there
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

					for _, address := range expectedBalances {
						// check balances
						var bal []models.BalanceRow
						q, err := di.Instance.DB.PrepareNamed(`select * from tracelistener.balances where address=:address`)
						require.NoError(t, err)
						defer q.Close()

						require.NoError(t,
							q.Select(
								&bal,
								map[string]interface{}{
									"address": address,
								},
							),
						)
						require.NotZero(t, len(bal))
						require.NotNil(t, bal)
					}

					for _, delegation := range expectedDelegations {
						// check delegations
						var del []models.DelegationRow
						d, err := di.Instance.DB.PrepareNamed(`select * from tracelistener.delegations where delegator_address=:delegator_address`)
						require.NoError(t, err)
						defer d.Close()
						require.NoError(t,
							d.Select(
								&del,
								map[string]interface{}{
									"delegator_address": delegation,
								},
							),
						)
						require.NotZero(t, len(del))
						require.NotNil(t, del[0].Delegator)
						require.NotZero(t, del[0].Amount)
					}

					for _, validator := range expectedValidators {
						// check validtaors
						var val []models.ValidatorRow
						d, err := di.Instance.DB.PrepareNamed(`select * from tracelistener.validators where operator_address=:validator_address`)
						require.NoError(t, err)
						defer d.Close()
						require.NoError(t,
							d.Select(
								&val,
								map[string]interface{}{
									"validator_address": validator,
								},
							),
						)

						require.NotZero(t, len(val))
						require.NotNil(t, val)
					}

					// PSA:
					// This code is commented because our current test store snapshot doesn't
					// have IBC data.
					// We can fix this in the future, but if all the other test pass this means
					// the bulk importer is working properly.

					// // check ibc connections
					// var conn []models.IBCConnectionRow
					// require.NoError(t,
					// 	di.Instance.Exec(
					// 		`select * from tracelistener.connections where client_id='07-tendermint-0'`,
					// 		nil,
					// 		&conn,
					// 	),
					// )
					// require.NotZero(t, len(conn))
					// require.NotNil(t, conn)

					// // check ibc clients
					// var cli []models.IBCClientStateRow
					// require.NoError(t,
					// 	di.Instance.Exec(
					// 		`select * from tracelistener.clients where client_id='07-tendermint-0'`,
					// 		nil,
					// 		&cli,
					// 	),
					// )
					// require.NotZero(t, len(cli))
					// require.NotNil(t, cli)
				}
			}
		})
	}
}
