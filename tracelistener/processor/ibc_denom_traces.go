package processor

import (
	"bytes"
	"sync"

	models "github.com/emerishq/demeris-backend-models/tracelistener"

	"github.com/emerishq/tracelistener/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener/processor/datamarshaler"
	"github.com/emerishq/tracelistener/tracelistener/tables"
	"go.uber.org/zap"
)

var denomTracesTable = tables.NewDenomTracesTable("tracelistener.denom_traces")

type ibcDenomTracesProcessor struct {
	l                *zap.SugaredLogger
	denomTracesCache map[string]models.IBCDenomTraceRow
	m                sync.Mutex
}

var (
	denomTracePathIndex = `CREATE INDEX IF NOT EXISTS denom_traces_path_idx ON ` + denomTracesTable.Name() + `(path)`
)

func (*ibcDenomTracesProcessor) Migrations() []string {
	return []string{
		denomTracesTable.CreateTable(),
		denomTracePathIndex,
	}
}

func (b *ibcDenomTracesProcessor) ModuleName() string {
	return "ibc_denom_traces"
}

func (b *ibcDenomTracesProcessor) SDKModuleName() tracelistener.SDKModuleName {
	return tracelistener.Transfer
}

func (b *ibcDenomTracesProcessor) UpsertStatement() string {
	return denomTracesTable.Upsert()
}

func (b *ibcDenomTracesProcessor) InsertStatement() string {
	return denomTracesTable.Insert()
}

func (b *ibcDenomTracesProcessor) DeleteStatement() string {
	panic("ibc denom trace processor never deletes")
}

func (b *ibcDenomTracesProcessor) FlushCache() []tracelistener.WritebackOp {
	b.m.Lock()
	defer b.m.Unlock()

	if len(b.denomTracesCache) == 0 {
		return nil
	}

	l := make([]models.DatabaseEntrier, 0, len(b.denomTracesCache))

	for _, c := range b.denomTracesCache {
		l = append(l, c)
	}

	b.denomTracesCache = map[string]models.IBCDenomTraceRow{}

	return []tracelistener.WritebackOp{
		{
			Type: tracelistener.Write,
			Data: l,
		},
	}
}

func (b *ibcDenomTracesProcessor) OwnsKey(key []byte) bool {
	return bytes.HasPrefix(key, datamarshaler.IBCDenomTracesKey)
}

func (b *ibcDenomTracesProcessor) Process(data tracelistener.TraceOperation) error {
	b.m.Lock()
	defer b.m.Unlock()

	res, err := datamarshaler.NewDataMarshaler(b.l).IBCDenomTraces(data)
	if err != nil {
		return err
	}

	b.denomTracesCache[res.Hash] = res

	return nil
}
