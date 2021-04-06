package tracelistener

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"go.uber.org/zap"
)

type WritebackOp struct {
	DatabaseExec string
	Data         []interface{}
}

// Operation is a kind of operations a TraceWatcher observes.
type Operation []byte

var (
	// WriteOp is a write trace operation
	WriteOp Operation = []byte("write")

	// DeleteOp is a write trace operation
	DeleteOp Operation = []byte("delete")

	// ReadOp is a write trace operation
	ReadOp Operation = []byte("read")

	// IterRangeOp is a write trace operation
	IterRangeOp Operation = []byte("iterRange")
)

type DataProcessorInfos struct {
	OpsChan            chan TraceOperation
	WritebackChan      chan []WritebackOp
	DatabaseMigrations []string
}

// DataProcessorFunc is the type of function used to initialize a data processor.
type DataProcessorFunc func(logger *zap.SugaredLogger) (DataProcessorInfos, error)

// TraceWatcher watches DataSource for WatchedOps, sends observed data over DataChan.
// Any observing error will be sent over ErrorChan.
// If WatchedOps is nil, all store operations will be sent over DataChan.
type TraceWatcher struct {
	DataSource io.Reader
	WatchedOps []Operation
	DataChan   chan<- TraceOperation
	ErrorChan  chan<- error
	Logger     *zap.SugaredLogger
}

func (tr *TraceWatcher) Watch() {
	fr := bufio.NewReader(tr.DataSource)

	for {
		line, err := fr.ReadBytes('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				fr = bufio.NewReader(tr.DataSource)
				continue
			}

			tr.Logger.Panicw("fatal error when reading from data source buffered reader", "error", err)
		}

		if !tr.mustConsiderData(line) {
			continue
		}

		to := TraceOperation{}
		if err := json.Unmarshal(line, &to); err != nil {
			tr.ErrorChan <- fmt.Errorf("failed unmarshaling, %w", err)
			continue
		}

		if len(to.Value) == 0 {
			continue
		}

		go func() {
			tr.DataChan <- to
		}()
	}
}

func (tr *TraceWatcher) mustConsiderData(b []byte) bool {
	if tr.WatchedOps == nil {
		return true
	}

	for _, op := range tr.WatchedOps {
		if bytes.Contains(b, op) {
			return true
		}
	}

	return false
}
