package tracelistener_test

import (
	"context"
	"encoding/json"
	"os"
	"syscall"
	"testing"

	"github.com/allinbits/tracelistener/tracelistener"
	"github.com/containerd/fifo"
	"go.uber.org/zap"
)

func BenchmarkTraceListener(b *testing.B) {
	f, err := os.CreateTemp("", "test_data")
	if err != nil {
		panic(err)
	}

	err = f.Close()
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
		err := loadTest(i, f.Name())
		if err != nil {
			panic(err)
		}
	}
}

func loadTest(height int, file string) error {
	ff, err := fifo.OpenFifo(context.Background(), file, syscall.O_WRONLY, 0655)
	if err != nil {
		return err
	}
	trace := tracelistener.TraceOperation{
		Operation:   string(tracelistener.WriteOp),
		Key:         []byte{0x68, 0x65, 0x6c, 0x6c, 0x6f, 0xa},
		Value:       []byte{0x68, 0x65, 0x6c, 0x6c, 0x6f, 0xa},
		BlockHeight: uint64(height),
		TxHash:      "A5CF62609D62ADDE56816681B6191F5F0252D2800FC2C312EB91D962AB7A97CB",
	}
	data, err := json.Marshal(trace)
	if err != nil {
		return err
	}
	ff.Write(data)
	err = ff.Close()
	if err != nil {
		return err
	}
	return nil
}
