package processor

import (
	"bytes"
	"sync"

	models "github.com/emerishq/demeris-backend-models/tracelistener"

	"github.com/emerishq/tracelistener/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener/processor/datamarshaler"
	"go.uber.org/zap"
)

type ibcDenomTracesProcessor struct {
	l                *zap.SugaredLogger
	denomTracesCache map[string]models.IBCDenomTraceRow
	m                sync.Mutex
}

func (*ibcDenomTracesProcessor) TableSchema() string {
	return createDenomTracesTable
}

func (b *ibcDenomTracesProcessor) ModuleName() string {
	return "ibc_denom_traces"
}

func (b *ibcDenomTracesProcessor) SDKModuleName() tracelistener.SDKModuleName {
	return tracelistener.Transfer
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
			DatabaseExec: insertDenomTrace,
			Data:         l,
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
