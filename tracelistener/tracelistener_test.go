package tracelistener_test

import (
	"fmt"
	"io"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/allinbits/demeris-backend/models"

	"github.com/allinbits/demeris-backend/tracelistener"
	"github.com/stretchr/testify/require"
)

func TestOperation_String(t *testing.T) {
	require.Equal(t, "write", tracelistener.WriteOp.String())
}

type testDatabaseEntrier struct {
	cn string
}

func (t testDatabaseEntrier) WithChainName(cn string) models.DatabaseEntrier {
	t.cn = cn
	return t
}

func TestWritebackOp_InterfaceSlice(t *testing.T) {
	tests := []struct {
		name   string
		fields []models.DatabaseEntrier
		want   []interface{}
	}{
		{
			"slice with single objects are equal",
			[]models.DatabaseEntrier{
				testDatabaseEntrier{cn: "cn"},
			},
			[]interface{}{
				testDatabaseEntrier{cn: "cn"},
			},
		},
		{
			"slice with multiple objects are equal",
			[]models.DatabaseEntrier{
				testDatabaseEntrier{cn: "cn"},
				testDatabaseEntrier{cn: "cn2"},
			},
			[]interface{}{
				testDatabaseEntrier{cn: "cn"},
				testDatabaseEntrier{cn: "cn2"},
			},
		},
		{
			"empty slice yields an empty one",
			[]models.DatabaseEntrier{},
			[]interface{}{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wb := tracelistener.WritebackOp{
				Data: tt.fields,
			}

			require.Equal(t, tt.want, wb.InterfaceSlice())
		})
	}
}

func TestTraceWatcher_Watch(t *testing.T) {
	tests := []struct {
		name        string
		ops         []tracelistener.Operation
		data        string
		wantErr     bool
		differentOp bool
		shouldPanic bool
		errSent     error
	}{
		{
			"write operation is configured and read accordingly",
			[]tracelistener.Operation{
				tracelistener.WriteOp,
			},
			writeOp,
			false,
			false,
			false,
			nil,
		},
		{
			"write operation is not configured and not read",
			[]tracelistener.Operation{
				tracelistener.ReadOp,
			},
			writeOp,
			false,
			true,
			false,
			nil,
		},
		{
			"any operation is configured and read accordingly",
			[]tracelistener.Operation{},
			writeOp,
			false,
			false,
			false,
			nil,
		},
		{
			"an EOF doesn't impact anything",
			[]tracelistener.Operation{},
			writeOp,
			false,
			false,
			false,
			io.EOF,
		},
		{
			"a random error panics",
			[]tracelistener.Operation{},
			writeOp,
			true,
			false,
			true,
			fmt.Errorf("error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			read, write := io.Pipe()

			dataChan := make(chan tracelistener.TraceOperation)
			errChan := make(chan error)
			l, _ := zap.NewDevelopment()
			tw := tracelistener.TraceWatcher{
				DataSource: read,
				WatchedOps: tt.ops,
				DataChan:   dataChan,
				ErrorChan:  errChan,
				Logger:     l.Sugar(),
			}

			go func() {
				if tt.shouldPanic {
					require.Panics(t, func() {
						tw.Watch()
					})
				} else {
					tw.Watch()
				}
			}()

			if tt.errSent != nil {
				require.NoError(t, write.CloseWithError(tt.errSent))
				time.Sleep(1 * time.Second)

				if !tt.shouldPanic {
					read, write = io.Pipe()
					tw.DataSource = read
				}
			}

			n, err := write.Write([]byte(fmt.Sprintf("%s\n", tt.data)))

			if !tt.shouldPanic {
				require.NoError(t, err)
				require.Equal(t, len(tt.data)+1, n)

				if tt.wantErr {
					require.Error(t, <-errChan)
					return
				}

				if !tt.differentOp {
					require.Eventually(t, func() bool {
						d := <-dataChan
						return d.Key != nil
					}, time.Second, 10*time.Millisecond)
					return
				}

				require.Never(t, func() bool {
					d := <-dataChan
					return d.Key != nil
				}, time.Second, 10*time.Millisecond)
			}
		})
	}
}
