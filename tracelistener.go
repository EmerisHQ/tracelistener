package tracelistener

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/allinbits/tracelistener/config"

	"go.uber.org/zap"
)

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

// BasicDatabaseEntry contains a list of all the fields each database row must contain in order to be
// inserted correctly.
type BasicDatabaseEntry struct {
	ChainName string `db:"chain_name"`
}

// DatabaseEntrier is implemented by each object that wants to be inserted in a database.
// It is usually used in conjunction to BasicDatabaseEntry.
type DatabaseEntrier interface {
	// WithChainName sets the ChainName field of the BasicDatabaseEntry struct.
	WithChainName(cn string) DatabaseEntrier
}

// WritebackOp represents a unit of database writeback operated by a processor.
// It contains the database query to be executed along with a slice of DatabaseEntrier data.
type WritebackOp struct {
	DatabaseExec string
	Data         []DatabaseEntrier
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
