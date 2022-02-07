package tracelistener_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	models "github.com/allinbits/demeris-backend-models/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener"
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
	op := `{"operation":"write","key":"aWJjL2Z3ZC8weGMwMDA0ZThkMzg=","value":"cG9ydHMvdHJhbnNmZXI=","metadata":null}`
	tests := []struct {
		name        string
		ops         []tracelistener.Operation
		data        string
		wantErr     bool
		differentOp bool
		shouldPanic bool
	}{
		{
			"write operation is configured and read accordingly",
			[]tracelistener.Operation{
				tracelistener.WriteOp,
			},
			op,
			false,
			false,
			false,
		},
		{
			"write operation is not configured and not read",
			[]tracelistener.Operation{
				tracelistener.ReadOp,
			},
			op,
			false,
			true,
			false,
		},
		{
			"any operation is configured and read accordingly",
			[]tracelistener.Operation{},
			op,
			false,
			false,
			false,
		},
		{
			"an EOF doesn't impact anything",
			[]tracelistener.Operation{},
			op,
			false,
			false,
			false,
		},
		{
			"a random error panics",
			[]tracelistener.Operation{},
			op,
			true,
			false,
			true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, err := os.CreateTemp("", "test_data")
			require.NoError(t, err)

			defer os.Remove(f.Name())

			dataChan := make(chan tracelistener.TraceOperation)
			errChan := make(chan error)
			l, _ := zap.NewDevelopment()
			tw := tracelistener.TraceWatcher{
				DataSourcePath: f.Name(),
				WatchedOps:     tt.ops,
				DataChan:       dataChan,
				ErrorChan:      errChan,
				Logger:         l.Sugar(),
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

			n, err := f.Write([]byte(tt.data))
			require.NoError(t, err)

			if !tt.shouldPanic {
				require.NoError(t, err)
				require.Equal(t, len(tt.data), n)

				if tt.wantErr {
					require.Error(t, <-errChan)
					return
				}

				if !tt.differentOp {
					require.Eventually(t, func() bool {
						d := <-dataChan
						return d.Key != nil
					}, 10*time.Second, 10*time.Millisecond)
					return
				}

				require.Never(t, func() bool {
					d := <-dataChan
					return d.Key != nil
				}, 10*time.Second, 10*time.Millisecond)
			}
		})
	}
}
