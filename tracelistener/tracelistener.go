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

type DataProcessorInfos struct {
	OpsChan            chan TraceOperation
	WritebackChan      chan []WritebackOp
	DatabaseMigrations []string
}

// DataProcessorFunc is the type of function used to initialize a data processor.
type DataProcessorFunc func(logger *zap.SugaredLogger, cfg *config.Config) (DataProcessorInfos, error)

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

		if to.Operation == WriteOp.String() && len(to.Value) == 0 {
			tr.Logger.Debugw("not considering data", "operation", to.Operation)
			continue
		}

		go func() {
			tr.DataChan <- to
		}()
	}
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
