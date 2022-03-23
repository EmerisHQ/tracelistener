package tracelistener

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	models "github.com/emerishq/demeris-backend-models/tracelistener"
	"github.com/nxadm/tail"

	"github.com/emerishq/tracelistener/tracelistener/config"

	"go.uber.org/zap"
)

// SDKModuleName represent a Cosmos SDK module name, available under the 'x' directory in the SDK
// codebase.
type SDKModuleName string

// String implements the fmt.Stringer interface.
func (smn SDKModuleName) String() string {
	return string(smn)
}

const (
	// Bank SDK module
	Bank SDKModuleName = "bank"

	// IBC SDK module
	IBC SDKModuleName = "ibc"

	// Staking SDK module
	Staking SDKModuleName = "staking"

	// Distribution SDK module
	Distribution SDKModuleName = "distribution"

	// IBC Transfer SDK module
	Transfer SDKModuleName = "transfer"

	// Account storage SDK module
	Acc SDKModuleName = "acc"
)

// SupportedSDKModuleList holds all the Cosmos SDK module names tracelistener supports.
var SupportedSDKModuleList = map[SDKModuleName]struct{}{
	Bank:         {},
	IBC:          {},
	Staking:      {},
	Distribution: {},
	Transfer:     {},
	Acc:          {},
}

const (
	// Info: https://github.com/cockroachdb/cockroach/issues/49256
	dbPlaceholderTotalLimit = 65535

	// we divide crdb placeholders 10x the maximum size to avoid
	// database retry congestion
	dbPlaceholderLimit = dbPlaceholderTotalLimit / 10
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

//go:generate stringer -type WritebackStatementTypes
type WritebackStatementTypes uint

const (
	Delete WritebackStatementTypes = iota
	Write
)

// WritebackOp represents a unit of database writeback operated by a processor.
// It contains the database query to be executed along with a slice of DatabaseEntrier data.
type WritebackOp struct {
	Type      WritebackStatementTypes
	Data      []models.DatabaseEntrier
	Statement string

	// SourceModule indicates the SDK module which initiated a WritebackOp.
	// It is used in bulk importing only.
	SourceModule string
}

// InterfaceSlice returns Data as a slice of interface{}.
func (wo WritebackOp) InterfaceSlice() []interface{} {
	dataIface := make([]interface{}, 0, len(wo.Data))
	for _, d := range wo.Data {
		dataIface = append(dataIface, d)
	}

	return dataIface
}

// DBPlaceholderAmount returns the total amount of database placeholders
// in wo.Data.
func (wo WritebackOp) DBPlaceholderAmount() int64 {
	return int64(int(wo.DBSinglePlaceholderAmount()) * len(wo.Data))
}

// DBSinglePlaceholderAmount returns the amount of struct fields of a single
// object in wo.Data.
func (wo WritebackOp) DBSinglePlaceholderAmount() int64 {
	fieldsAmount := reflect.TypeOf(wo.Data[0]).NumField()
	return int64(fieldsAmount)
}

// Split returns a slice of WritebackOps chunks, each containing a slice of
// DatabaseEntrier of maximum size chunkSize.
func (wo WritebackOp) Split(chunkSize int) []WritebackOp {
	ret := make([]WritebackOp, 0, chunkSize)
	for _, chunk := range buildEntrierChunks(wo.Data, int64(chunkSize)) {
		ret = append(ret, WritebackOp{
			Type:         wo.Type,
			Statement:    wo.Statement,
			Data:         chunk,
			SourceModule: wo.SourceModule,
		})
	}

	return ret
}

// Taken from: https://freshman.tech/snippets/go/split-slice-into-chunks/
// for laziness.
func buildEntrierChunks(slice []models.DatabaseEntrier, chunkSize int64) [][]models.DatabaseEntrier {
	preallocSize := len(slice) / int(chunkSize)
	chunks := make([][]models.DatabaseEntrier, 0, preallocSize)
	for i := int64(0); i < int64(len(slice)); i += chunkSize {
		end := i + chunkSize

		// necessary check to avoid slicing beyond
		// slice capacity
		if end > int64(len(slice)) {
			end = int64(len(slice))
		}

		chunks = append(chunks, slice[i:end])
	}

	return chunks
}

type DataProcessor interface {
	OpsChan() chan TraceOperation
	WritebackChan() chan []WritebackOp
	ErrorsChan() chan error
	DatabaseMigrations() []string
	Flush() error
	SetDBUpsertEnabled(enabled bool)
	StartBackgroundProcessing()
	StopBackgroundProcessing()
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
			break
		}

		for line := range t.Lines {
			if line.Err != nil {
				tr.ErrorChan <- fmt.Errorf("line reading error, line %v, error %w", line, err)
				break // restart the reading loop
			}

			tr.Logger.Debugw("new line read from reader", "line", line.Text)

			lineBytes := []byte(line.Text)

			// Log line used to trigger Grafana alerts.
			// Do not modify or remove without changing the corresponding dashboards
			tr.Logger.Infow("Probe", "c", "trace", "s", len(lineBytes))

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

			tr.Logger.Infow("trace processed",
				"kind", to.Operation,
				"block_height", to.BlockHeight,
				"tx_hash", to.TxHash,
			)
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
