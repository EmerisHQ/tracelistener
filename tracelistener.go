package tracelistener

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	config2 "github.com/allinbits/tracelistener/config"
	models "github.com/allinbits/demeris-backend-models/tracelistener"
	"github.com/nxadm/tail"

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
type DataProcessorFunc func(logger *zap.SugaredLogger, cfg *config2.Config) (DataProcessor, error)

// TraceWatcher watches DataSource for WatchedOps, sends observed data over DataChan.
// Any observing error will be sent over ErrorChan.
// If WatchedOps is nil, all store operations will be sent over DataChan.
type TraceWatcher struct {
	DataSourcePath string
	WatchedOps     []Operation
	DataChan       chan<- TraceOperation
	ErrorChan      chan<- error
	Logger         *zap.SugaredLogger
}

func (tr *TraceWatcher) Watch() {
	errorHappened := false
	for { // infinite cycle, if something goes wrong in reading the fifo we restart the cycle
		if errorHappened {
			// if a reading error happened, don't blast the cpu with retries,
			// wait some time then continue.
			errorHappened = false
			time.Sleep(250 * time.Millisecond)
		}

		t, err := tail.TailFile(
			tr.DataSourcePath, tail.Config{Follow: true, ReOpen: true, Pipe: true, Logger: tail.DiscardingLogger})
		if err != nil {
			tr.ErrorChan <- fmt.Errorf("tail creation error, %w", err)
			errorHappened = true
			break
		}

		for line := range t.Lines {
			if line.Err != nil {
				tr.ErrorChan <- fmt.Errorf("line reading error, line %v, error %w", line, err)
				errorHappened = true
				break // restart the reading loop
			}

			tr.Logger.Debugw("new line read from reader", "line", line.Text)

			lineBytes := []byte(line.Text)

			if !tr.mustConsiderData(lineBytes) {
				continue
			}

			to := TraceOperation{}
			if err := json.Unmarshal(lineBytes, &to); err != nil {
				tr.ErrorChan <- fmt.Errorf("failed unmarshaling, %w, data: %s", err, line.Text)
				continue
			}

			if err := tr.ParseOperation(to); err != nil {
				tr.ErrorChan <- fmt.Errorf("failed parsing operation, %w, data: %s", err, line.Text)
				continue
			}
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
