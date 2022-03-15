package bulk_test

import (
	"testing"

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

					addr := "69e2e6218dc4e610453afdb802c0e54cbadf6b49"
					// check balances
					var bal []models.BalanceRow
					q, err := di.Instance.DB.PrepareNamed(`select * from tracelistener.balances where address=:address`)
					require.NoError(t, err)
					defer q.Close()

					require.NoError(t,
						q.Select(
							&bal,
							map[string]interface{}{
								"address": addr,
							},
						),
					)
					require.NotZero(t, len(bal))
					require.NotNil(t, bal)

					// check delegations
					var del []models.DelegationRow
					d, err := di.Instance.DB.PrepareNamed(`select * from tracelistener.delegations where delegator_address=:delegator_address`)
					require.NoError(t, err)
					defer d.Close()
					require.NoError(t,
						d.Select(
							&del,
							map[string]interface{}{
								"delegator_address": addr,
							},
						),
					)
					require.NotZero(t, len(del))
					require.NotNil(t, del[0].Delegator)
					require.NotZero(t, del[0].Amount)

					// check unbonding_delegations
					var unDel []models.UnbondingDelegationRow
					ud, err := di.Instance.DB.PrepareNamed(`select * from tracelistener.unbonding_delegations where delegator_address=:delegator_address`)
					require.NoError(t, err)
					defer ud.Close()
					require.NoError(t,
						ud.Select(
							&unDel,
							map[string]interface{}{
								"delegator_address": addr,
							},
						),
					)
					require.NotZero(t, len(unDel))
					require.NotNil(t, unDel)

					// check validtaors
					var val []models.ValidatorRow
					require.NoError(t,
						di.Instance.Exec(
							`select * from tracelistener.validators where operator_address='cosmosvaloper1fkgp476xp2rhv8jjsyspl577v5emmz0ycftwez'`,
							nil,
							&val,
						),
					)
					require.NotZero(t, len(val))
					require.NotNil(t, val)

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
