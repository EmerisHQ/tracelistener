package tracelistener_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/emerishq/tracelistener/exporter"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	models "github.com/emerishq/demeris-backend-models/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener/database"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
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
			f, err := os.CreateTemp("", "test_data")
			require.NoError(t, err)

			defer func() { _ = os.Remove(f.Name()) }()

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
						tw.Watch(nil)
					})
				} else {
					tw.Watch(nil)
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
					}, 1*time.Second, 10*time.Millisecond)
					return
				}

				require.Never(t, func() bool {
					d := <-dataChan
					return d.Key != nil
				}, 1*time.Second, 10*time.Millisecond)
			}
		})
	}
}

func TestWritebackOp_SplitStatements(t *testing.T) {
	tests := []struct {
		name           string
		needle         tracelistener.WritebackOp
		limit          int
		expectedAmount int64
		mustPanic      bool
	}{
		{
			"limit equal to (fieldsAmount*4 - 1), returns 2 elements",
			tracelistener.WritebackOp{
				Type: tracelistener.Write,
				Data: []models.DatabaseEntrier{
					models.AuthRow{
						TracelistenerDatabaseRow: models.TracelistenerDatabaseRow{
							ChainName: "chain",
						},
						Address:        "address",
						SequenceNumber: 1,
						AccountNumber:  1,
					},
					models.AuthRow{
						TracelistenerDatabaseRow: models.TracelistenerDatabaseRow{
							ChainName: "chain",
						},
						Address:        "address",
						SequenceNumber: 1,
						AccountNumber:  1,
					},
					models.AuthRow{
						TracelistenerDatabaseRow: models.TracelistenerDatabaseRow{
							ChainName: "chain",
						},
						Address:        "address",
						SequenceNumber: 1,
						AccountNumber:  1,
					},
					models.AuthRow{
						TracelistenerDatabaseRow: models.TracelistenerDatabaseRow{
							ChainName: "chain",
						},
						Address:        "address",
						SequenceNumber: 1,
						AccountNumber:  1,
					},
				},
			},
			15,
			2,
			false,
		},
		{
			"limit of fieldsAmount returns exactly len(needle.Data)",
			tracelistener.WritebackOp{
				Type: tracelistener.Write,
				Data: []models.DatabaseEntrier{
					models.AuthRow{
						TracelistenerDatabaseRow: models.TracelistenerDatabaseRow{
							ChainName: "chain",
						},
						Address:        "address",
						SequenceNumber: 1,
						AccountNumber:  1,
					},
					models.AuthRow{
						TracelistenerDatabaseRow: models.TracelistenerDatabaseRow{
							ChainName: "chain",
						},
						Address:        "address",
						SequenceNumber: 1,
						AccountNumber:  1,
					},
					models.AuthRow{
						TracelistenerDatabaseRow: models.TracelistenerDatabaseRow{
							ChainName: "chain",
						},
						Address:        "address",
						SequenceNumber: 1,
						AccountNumber:  1,
					},
					models.AuthRow{
						TracelistenerDatabaseRow: models.TracelistenerDatabaseRow{
							ChainName: "chain",
						},
						Address:        "address",
						SequenceNumber: 1,
						AccountNumber:  1,
					},
				},
			},
			4,
			4,
			false,
		},
		{
			"limit greater than fieldsAmount*4 returns exactly 1 element",
			tracelistener.WritebackOp{
				Type: tracelistener.Write,
				Data: []models.DatabaseEntrier{
					models.AuthRow{
						TracelistenerDatabaseRow: models.TracelistenerDatabaseRow{
							ChainName: "chain",
						},
						Address:        "address",
						SequenceNumber: 1,
						AccountNumber:  1,
					},
					models.AuthRow{
						TracelistenerDatabaseRow: models.TracelistenerDatabaseRow{
							ChainName: "chain",
						},
						Address:        "address",
						SequenceNumber: 1,
						AccountNumber:  1,
					},
					models.AuthRow{
						TracelistenerDatabaseRow: models.TracelistenerDatabaseRow{
							ChainName: "chain",
						},
						Address:        "address",
						SequenceNumber: 1,
						AccountNumber:  1,
					},
					models.AuthRow{
						TracelistenerDatabaseRow: models.TracelistenerDatabaseRow{
							ChainName: "chain",
						},
						Address:        "address",
						SequenceNumber: 1,
						AccountNumber:  1,
					},
				},
			},
			40,
			1,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			panicf := require.Panics
			if !tt.mustPanic {
				panicf = require.NotPanics
			}

			val := []tracelistener.WritebackOp{}
			panicf(t, func() {
				val = tt.needle.SplitStatements(tt.limit)
			})

			require.Len(t, val, int(tt.expectedAmount))
		})
	}
}

func TestWritebackOp_DBPlaceholderAmount(t *testing.T) {
	tests := []struct {
		name string
		data []models.DatabaseEntrier
		want int64
	}{
		{
			"1 databaseentrier with fields amount = 4, return 4",
			[]models.DatabaseEntrier{
				models.AuthRow{
					TracelistenerDatabaseRow: models.TracelistenerDatabaseRow{
						ChainName: "chain",
					},
					Address:        "address",
					SequenceNumber: 1,
					AccountNumber:  1,
				},
			},
			4,
		},
		{
			"4 databaseentrier with fields amount = 4, return 16",
			[]models.DatabaseEntrier{
				models.AuthRow{
					TracelistenerDatabaseRow: models.TracelistenerDatabaseRow{
						ChainName: "chain",
					},
					Address:        "address",
					SequenceNumber: 1,
					AccountNumber:  1,
				},
				models.AuthRow{
					TracelistenerDatabaseRow: models.TracelistenerDatabaseRow{
						ChainName: "chain",
					},
					Address:        "address",
					SequenceNumber: 1,
					AccountNumber:  1,
				},
				models.AuthRow{
					TracelistenerDatabaseRow: models.TracelistenerDatabaseRow{
						ChainName: "chain",
					},
					Address:        "address",
					SequenceNumber: 1,
					AccountNumber:  1,
				},
				models.AuthRow{
					TracelistenerDatabaseRow: models.TracelistenerDatabaseRow{
						ChainName: "chain",
					},
					Address:        "address",
					SequenceNumber: 1,
					AccountNumber:  1,
				},
			},
			16,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wo := tracelistener.WritebackOp{
				Type: tracelistener.Write,
				Data: tt.data,
			}

			require.Equal(t, tt.want, wo.DBPlaceholderAmount())
		})
	}
}

func TestWritebackOp_DBSinglePlaceholderAmount(t *testing.T) {
	tests := []struct {
		name string
		data []models.DatabaseEntrier
		want int64
	}{
		{
			"databaseentrier with fields amount = 4, return 4",
			[]models.DatabaseEntrier{
				models.AuthRow{
					TracelistenerDatabaseRow: models.TracelistenerDatabaseRow{
						ChainName: "chain",
					},
					Address:        "address",
					SequenceNumber: 1,
					AccountNumber:  1,
				},
			},
			4,
		},
		{
			"4 databaseentrier with fields amount = 4, return 4",
			[]models.DatabaseEntrier{
				models.AuthRow{
					TracelistenerDatabaseRow: models.TracelistenerDatabaseRow{
						ChainName: "chain",
					},
					Address:        "address",
					SequenceNumber: 1,
					AccountNumber:  1,
				},
				models.AuthRow{
					TracelistenerDatabaseRow: models.TracelistenerDatabaseRow{
						ChainName: "chain",
					},
					Address:        "address",
					SequenceNumber: 1,
					AccountNumber:  1,
				},
				models.AuthRow{
					TracelistenerDatabaseRow: models.TracelistenerDatabaseRow{
						ChainName: "chain",
					},
					Address:        "address",
					SequenceNumber: 1,
					AccountNumber:  1,
				},
				models.AuthRow{
					TracelistenerDatabaseRow: models.TracelistenerDatabaseRow{
						ChainName: "chain",
					},
					Address:        "address",
					SequenceNumber: 1,
					AccountNumber:  1,
				},
			},
			4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wo := tracelistener.WritebackOp{
				Type: tracelistener.Write,
				Data: tt.data,
			}

			require.Equal(t, tt.want, wo.DBSinglePlaceholderAmount())
		})
	}
}

func TestWritebackOp_SplitStatementToDBLimit(t *testing.T) {
	unit := models.AuthRow{
		TracelistenerDatabaseRow: models.TracelistenerDatabaseRow{
			ChainName: "chain",
		},
		Address:        "address",
		SequenceNumber: 1,
		AccountNumber:  1,
	}

	wu := tracelistener.WritebackOp{
		Type: tracelistener.Write,
		Data: make([]models.DatabaseEntrier, 16385),
	}

	// building a writebackop with data which goes past the postgresql placeholder amount
	// 16385 * 4 (AuthRow) = 65540
	for i := 0; i < 16385; i++ {
		wu.Data[i] = unit
	}

	out := wu.SplitStatementToDBLimit()
	require.Len(t, out, 12, "expected len=12, got %d", len(out))
}

type InsertType struct {
	Field string `db:"field"`
}

// implement models.DatabaseEntrier on insertType
func (i InsertType) WithChainName(cn string) models.DatabaseEntrier {
	// no-op
	return i
}

func TestWritebackOp_ChunkingWorks(t *testing.T) {
	ts, err := testserver.NewTestServer()
	require.NoError(t, err)
	require.NoError(t, ts.WaitForInit())
	defer func() {
		ts.Stop()
	}()

	connString := ts.PGURL().String()

	i, err := database.New(connString)
	require.NoError(t, err)

	// fake database schema
	schema := `create table defaultdb.testtable (field text not null)`
	insert := `insert into defaultdb.testtable (field) values (:field)`

	insertData := make([]models.DatabaseEntrier, 0, 70000)
	for i := 0; i < 70000; i++ {
		insertData = append(insertData, InsertType{
			Field: strconv.Itoa(i),
		})
	}

	_, err = i.Instance.DB.Exec(schema)
	require.NoError(
		t,
		err,
	)

	dbe := tracelistener.WritebackOp{
		Type: tracelistener.Write,
		Data: insertData,
	}

	insertErr := i.Add(insert, dbe.InterfaceSlice())

	require.Error(t, insertErr)
	require.Contains(t, insertErr.Error(), "placeholder index must be between 1 and 65536", insertErr.Error())

	// check that the amount of statements after split is equal to
	// the amount before split
	splitStatements := dbe.SplitStatementToDBLimit()

	totalStatements := 0
	for _, ss := range splitStatements {
		totalStatements += len(ss.Data)
	}

	require.Equal(t, len(dbe.Data), totalStatements)

	// insert with chunking
	for _, chunk := range splitStatements {
		insertErr := i.Add(insert, chunk.InterfaceSlice())

		require.NoError(t, insertErr)
	}
}

func TestTracelistener_Exporter_invalidParams(t *testing.T) {
	op := `{"operation":"write","key":"aWJjL2Z3ZC8weGMwMDA0ZThkMzg=","value":"cG9ydHMvdHJhbnNmZXI=","metadata":null}`
	tests := []struct {
		name       string
		data       string
		params     string
		getRespMsg string
	}{
		{
			"no param: returns validation error",
			op,
			"",
			"invalid param combination",
		},
		{
			"duration invalid param: exceeds the range",
			op,
			"?duration=30h",
			fmt.Sprintf("validation error: accepted duration 1s-%s received %s", exporter.MaxDuration, (time.Hour * 30).String()),
		},
		{
			"duration invalid param: below the range",
			op,
			"?duration=-30h",
			fmt.Sprintf("validation error: accepted duration 1s-%s received %s", exporter.MaxDuration, (-time.Hour * 30).String()),
		},
		{
			"duration invalid param: wrong signature",
			op,
			"?D=30m",
			"validation error: unknown param D",
		},
		{
			"count invalid param: malformed, missing suffix `N`",
			op,
			fmt.Sprintf("?count=%d", exporter.MaxTraceCount+1),
			"invalid query param count, want format 20N got " + fmt.Sprint(exporter.MaxTraceCount+1),
		},
		{
			"count invalid param: exceeds the range",
			op,
			fmt.Sprintf("?count=%dN", exporter.MaxTraceCount+1),
			"validation error: accepted trace count 1-1000000 received " + fmt.Sprint(exporter.MaxTraceCount+1),
		},
		{
			"count invalid param: below the range",
			op,
			fmt.Sprintf("?count=%dN", -1),
			"validation error: accepted trace count 1-1000000 received -1",
		},
		{
			"count invalid param: wrong signature",
			op,
			fmt.Sprintf("?N=%dN", -1),
			"validation error: unknown param N",
		},
		{
			"size invalid param: malformed, missing suffix `MB`",
			op,
			fmt.Sprintf("?size=%d", exporter.MaxSizeLim),
			"invalid query param size, want format 20MB got " + fmt.Sprint(exporter.MaxSizeLim),
		},
		{
			"size invalid param: exceeds the range",
			op,
			fmt.Sprintf("?size=%dMB", exporter.MaxSizeLim+1),
			fmt.Sprintf("validation error: accepted record file size 1-%dMB received %d", exporter.MaxSizeLim, exporter.MaxSizeLim+1),
		},
		{
			"size invalid param: below the range",
			op,
			fmt.Sprintf("?size=%dMB", -1),
			fmt.Sprintf("validation error: accepted record file size 1-%dMB received %d", exporter.MaxSizeLim, -1),
		},
		{
			"size invalid param: wrong signature",
			op,
			fmt.Sprintf("?M=%dMB", 10),
			"validation error: unknown param M",
		},
		{
			"invalid param xxxx: returns validation error",
			op,
			"?count=10N&xxxx=yyyy",
			"validation error: unknown param xxxx",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.CreateTemp("", "test_data")
			require.NoError(t, err)

			t.Cleanup(func() {
				_ = os.Remove(f.Name())
			})

			dataChan := make(chan tracelistener.TraceOperation)
			errChan := make(chan error)
			l, _ := zap.NewDevelopment()
			tw := tracelistener.TraceWatcher{
				DataSourcePath: f.Name(),
				WatchedOps:     []tracelistener.Operation{},
				DataChan:       dataChan,
				ErrorChan:      errChan,
				Logger:         l.Sugar(),
			}

			exp, err := exporter.New(exporter.WithLogger(l.Sugar()))
			require.NoError(t, err)

			p, err := getFreePort(t)
			require.NoError(t, err)

			go exp.ListenAndServeHTTP(fmt.Sprintf("%d", p))
			go tw.Watch(exp)

			r, _ := http.Get(fmt.Sprintf("http://localhost:%d/start%s", p, tt.params))
			require.Eventually(t, func() bool {
				return r.Body != nil
			}, time.Second*15, time.Millisecond*100)
			by, err := ioutil.ReadAll(r.Body)
			require.NoError(t, err)
			require.NoError(t, r.Body.Close())
			require.Contains(t, string(by), tt.getRespMsg)

			r, err = http.Get(fmt.Sprintf("http://localhost:%d/stat", p))
			require.NoError(t, err)
			require.Eventually(t, func() bool {
				return r.Body != nil
			}, time.Second*5, time.Millisecond*100)
			by, err = ioutil.ReadAll(r.Body)
			require.Contains(t, string(by), exporter.ErrExporterNotRunning.Error())
			require.NoError(t, r.Body.Close())

			n, err := f.Write([]byte(tt.data))
			require.NoError(t, err)
			require.Equal(t, len(tt.data), n)
		})
	}
}

func TestTracelistener_Exporter_success(t *testing.T) {
	op := `{"operation":"write","key":"aWJjL2Z3ZC8weGMwMDA0ZThkMzg=","value":"cG9ydHMvdHJhbnNmZXI=","metadata":null}`
	tests := []struct {
		name          string
		N             int
		params        string
		generateTrace int
	}{
		{
			"Capture N traces",
			10,
			"?count=10N",
			20,
		},
		{
			"Capture N traces: N satisfied earliest",
			10,
			"?count=10N&size=100MB&duration=1h",
			200,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeFile := fmt.Sprintf("fifo%d.fifo", i)
			_ = os.Remove(pipeFile)
			require.NoError(t, syscall.Mkfifo(pipeFile, 0666))

			l, _ := zap.NewDevelopment()
			tw := tracelistener.TraceWatcher{
				DataSourcePath: pipeFile,
				WatchedOps:     []tracelistener.Operation{},
				DataChan:       make(chan tracelistener.TraceOperation),
				ErrorChan:      make(chan error),
				Logger:         l.Sugar(),
			}

			exp, err := exporter.New(exporter.WithLogger(l.Sugar()))
			require.NoError(t, err)

			port, err := getFreePort(t)
			require.NoError(t, err)

			go exp.ListenAndServeHTTP(fmt.Sprintf("%d", port))
			go tw.Watch(exp)

			r, _ := http.Get(fmt.Sprintf("http://localhost:%d/start%s", port, tt.params))
			var stat1 any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&stat1))
			ssGet, ok := stat1.(map[string]any)
			require.True(t, ok)

			t.Cleanup(func() {
				_ = os.Remove(pipeFile)
				_ = os.Remove(fmt.Sprint(ssGet["file_name"]))
			})
			require.NoError(t, r.Body.Close())

			r, err = http.Get(fmt.Sprintf("http://localhost:%d/stat", port))
			require.NoError(t, err)
			require.NoError(t, json.NewDecoder(r.Body).Decode(&stat1))
			ssStat, ok := stat1.(map[string]any)
			require.True(t, ok)
			require.NoError(t, r.Body.Close())

			// trace_count must be same as we haven't fed any traces yet.
			// file_name and start_time are thrown in for readers sanity.
			require.Equal(t, ssGet["file_name"], ssStat["file_name"])
			require.Equal(t, ssGet["start_time"], ssStat["start_time"])
			require.InDelta(t, ssGet["trace_count"], 0, 0)
			require.Equal(t, ssGet["trace_count"], ssStat["trace_count"])

			f, err := os.OpenFile(pipeFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
			require.NoError(t, err)

			// Simulate a chain. Feed data to the fifo. Which will be captured & processed
			// by TL. Also, tt.N traces should be captured by exporter.
			for i := 0; i < tt.generateTrace; i++ {
				n, err := f.WriteString(op + "\n")
				require.NoError(t, err)
				require.Equal(t, len(op)+1, n)
			}

			require.Eventually(t, func() bool {
				r, err = http.Get(fmt.Sprintf("http://localhost:%d/stat", port))
				require.NoError(t, err)
				var stat any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&stat))
				ssStat, ok := stat.(map[string]any)
				require.True(t, ok)
				require.NoError(t, r.Body.Close())
				traceCountFromStat, ok := ssStat["trace_count"].(float64)
				require.True(t, ok)
				return int(traceCountFromStat) == tt.N
			}, time.Second*10, time.Millisecond*200)
		})
	}
}

func getFreePort(t *testing.T) (int, error) {
	t.Helper()
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port, nil
}
