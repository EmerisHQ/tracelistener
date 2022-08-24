package tracelistener_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/allinbits/tracelistener/tracelistener"
	"go.uber.org/zap"
)

func BenchmarkTraceListener(b *testing.B) {
	f, err := os.CreateTemp("", "test_data")
	if err != nil {
		panic(err)
	}

	defer os.Remove(f.Name())

	dataChan := make(chan tracelistener.TraceOperation)
	errChan := make(chan error)
	l, _ := zap.NewDevelopment()
	tw := tracelistener.TraceWatcher{
		DataSourcePath: f.Name(),
		WatchedOps: []tracelistener.Operation{
			tracelistener.WriteOp,
			tracelistener.DeleteOp,
		},
		DataChan:  dataChan,
		ErrorChan: errChan,
		Logger:    l.Sugar(),
	}

	go func() {
		tw.Watch()
	}()

	for i := 0; i < b.N; i++ {
		loadTest(b, i, f)
	}

	err = f.Close()
	if err != nil {
		panic(err)
	}
}

func loadTest(b *testing.B, height int, file *os.File) {
	b.Helper()
	op := fmt.Sprintf(`{"operation":"write","key":"aGVsbG8K","value":"aGVsbG8K","metadata":{"blockHeight":%d,"txHash":"A5CF62609D62ADDE56816681B6191F5F0252D2800FC2C312EB91D962AB7A97CB"}}`, height)
	fmt.Fprintf(file, "%s\n", op)
}
