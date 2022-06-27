//go:build !race
// +build !race

package blocktime_test

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emerishq/tracelistener/models"
	"github.com/gorilla/websocket"

	"github.com/tendermint/tendermint/types"

	coretypes "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/emerishq/tracelistener/tracelistener/blocktime"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	"github.com/emerishq/emeris-utils/database"
	"github.com/stretchr/testify/require"

	"go.uber.org/zap"
)

func TestWatcher_ParseBlockData(t *testing.T) {
	tests := []struct {
		name        string
		blockData   coretypes.ResultEvent
		wantErr     bool
		checkInsert bool
	}{
		{
			"data is not EventDataNewBlock",
			coretypes.ResultEvent{
				Query:  "query",
				Data:   "test",
				Events: nil,
			},
			true,
			false,
		},
		{
			"data is EventDataNewBlock but block.Block is nil",
			coretypes.ResultEvent{
				Query: "query",
				Data: types.EventDataNewBlock{
					Block: nil,
				},
				Events: nil,
			},
			false,
			false,
		},
		{
			"data is EventDataNewBlock with valid time, which is inserted",
			coretypes.ResultEvent{
				Query: "query",
				Data: types.EventDataNewBlock{
					Block: &types.Block{
						Header: types.Header{
							Time: time.Now(),
						},
					},
				},
				Events: nil,
			},
			false,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, err := testserver.NewTestServer()
			require.NoError(t, err)
			require.NoError(t, ts.WaitForInit())
			defer func() {
				ts.Stop()
			}()

			connString := ts.PGURL().String()

			i, err := database.New(connString)

			require.NoError(t, database.RunMigrations(connString, []string{
				"CREATE DATABASE tracelistener;",
				blocktime.CreateTable,
			}))

			w := blocktime.New(
				i,
				"test",
				zap.NewNop().Sugar(),
			)

			err = w.ParseBlockData(tt.blockData)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.checkInsert {
				var blo []models.BlockTimeRow
				require.NoError(t,
					i.Exec(
						`select * from tracelistener.blocktime where chain_name='test'`,
						nil,
						&blo,
					),
				)

				require.Len(t, blo, 1)
				require.NotZero(t, blo[0].BlockTime.Unix())
			}
		})
	}
}

func TestWatcher_InsertBlockTime(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	tests := []struct {
		name      string
		blockData coretypes.ResultEvent
		blockTime time.Time
	}{
		{
			"old blocktime",
			coretypes.ResultEvent{
				Query: "query",
				Data: types.EventDataNewBlock{
					Block: &types.Block{
						Header: types.Header{
							Time: now.Add(time.Hour * -1),
						},
					},
				},
				Events: nil,
			},
			now,
		},
		{
			"same blocktime",
			coretypes.ResultEvent{
				Query: "query",
				Data: types.EventDataNewBlock{
					Block: &types.Block{
						Header: types.Header{
							Time: now,
						},
					},
				},
				Events: nil,
			},
			now,
		},
		{
			"later blocktime",
			coretypes.ResultEvent{
				Query: "query",
				Data: types.EventDataNewBlock{
					Block: &types.Block{
						Header: types.Header{
							Time: now.Add(time.Hour),
						},
					},
				},
				Events: nil,
			},
			now.Add(time.Hour),
		},
	}

	ts, err := testserver.NewTestServer()
	require.NoError(t, err)
	require.NoError(t, ts.WaitForInit())
	defer func() {
		ts.Stop()
	}()

	connString := ts.PGURL().String()

	I, err := database.New(connString)

	require.NoError(t, database.RunMigrations(connString, []string{
		"CREATE DATABASE tracelistener;",
		blocktime.CreateTable,
	}))

	w := blocktime.New(
		I,
		"test",
		zap.NewNop().Sugar(),
	)

	// Insert inital data
	err = w.ParseBlockData(coretypes.ResultEvent{
		Query: "query",
		Data: types.EventDataNewBlock{
			Block: &types.Block{
				Header: types.Header{
					Time: now,
				},
			},
		},
		Events: nil,
	})
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, w.ParseBlockData(tt.blockData))
			var blo []models.BlockTimeRow
			require.NoError(t,
				I.Exec(
					`select * from tracelistener.blocktime where chain_name='test'`,
					nil,
					&blo,
				),
			)

			require.Len(t, blo, 1)
			require.Equal(t, tt.blockTime, blo[0].BlockTime)
		})
	}
}

func TestNew(t *testing.T) {
	l := zap.NewNop().Sugar()
	cn := "chainName"
	i := &database.Instance{}

	require.NotNil(t, blocktime.New(i, cn, l))
}

type fakeWS struct {
	u        websocket.Upgrader
	failCall bool
}

func (fws *fakeWS) Handler(w http.ResponseWriter, r *http.Request) {
	if fws.failCall {
		http.Error(w, "bad", http.StatusBadRequest)
		return
	}

	c, err := fws.u.Upgrade(w, r, nil)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = c.Close()
	}()
}

func TestWatcher_Connect(t *testing.T) {
	tests := []struct {
		name      string
		ws        fakeWS
		chainName string
		wantErr   bool
	}{
		{
			"connection works",
			fakeWS{},
			"127.0.0.1",
			false,
		},
		{
			"connection doesn't work",
			fakeWS{},
			"fake",
			true,
		},
		{
			"connection works but the server returns 400",
			fakeWS{
				failCall: true,
			},
			"127.0.0.1",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := tendermintFakeServer(t, http.HandlerFunc(tt.ws.Handler))
			defer srv.Close()

			ts, err := testserver.NewTestServer()
			require.NoError(t, err)
			require.NoError(t, ts.WaitForInit())
			defer func() {
				ts.Stop()
			}()

			connString := ts.PGURL().String()

			i, err := database.New(connString)

			bt := blocktime.New(
				i,
				tt.chainName,
				zap.NewNop().Sugar(),
			)

			err = bt.Connect()

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

		})
	}
}

func tendermintFakeServer(t *testing.T, h http.Handler) *httptest.Server {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:26657")

	if err != nil {
		t.Fatal(fmt.Sprintf("httptest: failed to listen on 127.0.0.1:26657: %v", err))
	}

	s := &httptest.Server{
		Listener: l,
		Config:   &http.Server{Handler: h},
	}

	s.Start()

	return s
}
