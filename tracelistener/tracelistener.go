package tracelistener

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/allinbits/demeris-backend/models"

	"github.com/allinbits/demeris-backend/tracelistener/config"

	"go.uber.org/zap"
)

// Operation is a kind of operations a TraceWatcher observes.
type Operation []byte

// String implements fmt.Stringer on Operation.
func (o Operation) String() string {
	return string(o)
}

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

// WritebackOp represents a unit of database writeback operated by a processor.
// It contains the database query to be executed along with a slice of DatabaseEntrier data.
type WritebackOp struct {
	DatabaseExec string
	Data         []models.DatabaseEntrier
}

// InterfaceSlice returns Data as a slice of interface{}.
func (wo WritebackOp) InterfaceSlice() []interface{} {
	dataIface := make([]interface{}, 0, len(wo.Data))
	for _, d := range wo.Data {
		dataIface = append(dataIface, d)
	}

	return dataIface
}

type DataProcessor interface {
	OpsChan() chan TraceOperation
	WritebackChan() chan []WritebackOp
	ErrorsChan() chan error
	DatabaseMigrations() []string
	Flush() error
}

type TracingError struct {
	InnerError error
	Module     string
	Data       TraceOperation
}

func (t TracingError) Error() string {
	return fmt.Sprintf("%s: %s", t.Module, t.InnerError)
}

// DataProcessorFunc is the type of function used to initialize a data processor.
type DataProcessorFunc func(logger *zap.SugaredLogger, cfg *config.Config) (DataProcessor, error)

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

		tr.Logger.Debugw("new line read from reader", "line", string(line))

		if !tr.mustConsiderData(line) {
			continue
		}

		to := TraceOperation{}
		if err := json.Unmarshal(line, &to); err != nil {
			tr.ErrorChan <- fmt.Errorf("failed unmarshaling, %w, data: %s", err, string(line))
			continue
		}

		if err := tr.ParseOperation(to); err != nil {
			tr.ErrorChan <- fmt.Errorf("failed parsing operation, %w, data: %s", err, string(line))
			continue
		}
	}
}

func (tr *TraceWatcher) ParseOperation(data TraceOperation) error {
	if !tr.mustConsiderOperation(data) {
		return nil
	}

	if data.Operation == WriteOp.String() && len(data.Value) == 0 {
		tr.Logger.Debugw("not considering data", "operation", data.Operation)
		return nil
	}

	go func() {
		tr.DataChan <- data
	}()

	return nil
}

func (tr *TraceWatcher) mustConsiderData(b []byte) bool {
	if tr.WatchedOps == nil || len(tr.WatchedOps) == 0 {
		return true
	}

	for _, op := range tr.WatchedOps {
		if bytes.Contains(b, op) {
			return true
		}
	}

	return false
}

func (tr *TraceWatcher) mustConsiderOperation(op TraceOperation) bool {
	if tr.WatchedOps == nil || len(tr.WatchedOps) == 0 {
		return true
	}

	for _, wopts := range tr.WatchedOps {
		if wopts.String() == op.Operation {
			return true
		}
	}

	return false
}
