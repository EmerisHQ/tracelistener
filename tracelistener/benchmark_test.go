package tracelistener_test

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"syscall"
	"testing"

	"github.com/allinbits/tracelistener/tracelistener"
	"github.com/containerd/fifo"
	"go.uber.org/zap"
)

func setup(b *testing.B) (io.ReadWriteCloser, string) {
	b.Helper()
	f, err := os.CreateTemp("", "test_data")
	if err != nil {
		panic(err)
	}

	err = f.Close()
	if err != nil {
		panic(err)
	}

	dataChan := make(chan tracelistener.TraceOperation)
	errChan := make(chan error)
	l := zap.NewNop()
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
		// drain data channel
		for range dataChan {
		}
	}()

	go func() {
		tw.Watch()
	}()

	ff, err := fifo.OpenFifo(context.Background(), f.Name(), syscall.O_WRONLY, 0655)
	if err != nil {
		panic(err)
	}

	return ff, f.Name()
}

func runBenchmark(b *testing.B, amount int, kind string) {
	ff, fifoName := setup(b)

	b.ResetTimer()

	for i := 0; i < amount; i++ {
		err := loadTest(b, i, ff, kind)
		if err != nil {
			panic(err)
		}
	}

	os.Remove(fifoName)

	ff.Close()
}

func BenchmarkTracelistenerRealTraces(b *testing.B) {
	b.Log("reading test traces file...")
	lines, err := loadTestFile(b)
	if err != nil {
		b.Fatal(err)
	}
	b.Log("finished reading test traces file!")

	ff, fifoName := setup(b)

	b.ResetTimer()

	for _, line := range lines {
		fmt.Fprintf(ff, line+"\n")
	}

	os.Remove(fifoName)

	ff.Close()
}

func BenchmarkTraceListenerKindWrite(b *testing.B) {
	runBenchmark(b, b.N, "write")
}

func BenchmarkTraceListener100KKindWrite(b *testing.B) {
	runBenchmark(b, 100000, "write")
}

func BenchmarkTraceListener1MKindWrite(b *testing.B) {
	runBenchmark(b, 1000000, "write")
}

func BenchmarkTraceListener1MKindIterRange(b *testing.B) {
	runBenchmark(b, 1000000, "IterRange")
}

func BenchmarkTraceListener10MKindWrite(b *testing.B) {
	runBenchmark(b, 10000000, "write")
}

func loadTest(b *testing.B, height int, ff io.Writer, kind string) error {
	b.Helper()

	// trace := tracelistener.TraceOperation{
	// 	Operation:   string(tracelistener.WriteOp),
	// 	Key:         []byte{0x68, 0x65, 0x6c, 0x6c, 0x6f, 0xa},
	// 	Value:       []byte{0x68, 0x65, 0x6c, 0x6c, 0x6f, 0xa},
	// 	BlockHeight: uint64(height),
	// 	TxHash:      "A5CF62609D62ADDE56816681B6191F5F0252D2800FC2C312EB91D962AB7A97CB",
	// }
	// data, err := json.Marshal(trace)
	// if err != nil {
	// 	return err
	// }

	// println(string(data))

	s := `{"operation":"%s","key":"aGVsbG8K","value":"aGVsbG8K","block_height":158284,"tx_hash":"A5CF62609D62ADDE56816681B6191F5F0252D2800FC2C312EB91D962AB7A97CB","SuggestedProcessor":""}`

	fmt.Fprintf(ff, s+"\n", kind)

	return nil
}

func loadTestFile(b *testing.B) ([]string, error) {
	b.Helper()

	fname := os.Getenv("TRACELISTENER_BENCH_TRACEFILE")
	if fname == "" {
		return nil, fmt.Errorf("TRACELISTENER_BENCH_TRACEFILE environment variable not defined")
	}

	file, err := os.Open(fname)
	if err != nil {
		return nil, fmt.Errorf("cannot open file %s, %w", fname, err)
	}

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 1000000) // a very high capacity
	scanner.Buffer(buf, 1000000)

	ret := []string{}
	for scanner.Scan() {
		ret = append(ret, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning error, %w", err)
	}

	return ret, nil
}
