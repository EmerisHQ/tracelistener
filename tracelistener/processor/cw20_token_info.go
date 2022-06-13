package processor

import (
	"sync"

	"go.uber.org/zap"

	models "github.com/emerishq/demeris-backend-models/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener/processor/datamarshaler"
	"github.com/emerishq/tracelistener/tracelistener/tables"
)

var cw20TokenInfoTable = tables.NewCw20TokenInfoTable("tracelistener.cw20_token_infos")

type cw20TokenInfoCacheEntry struct {
	contractAddress string
}

type cw20TokenInfoProcessor struct {
	l           *zap.SugaredLogger
	heightCache map[cw20TokenInfoCacheEntry]models.CW20TokenInfoRow
	m           sync.Mutex
}

func (*cw20TokenInfoProcessor) Migrations() []string {
	return []string{cw20TokenInfoTable.CreateTable()}
}

func (b *cw20TokenInfoProcessor) ModuleName() string {
	return "cw20_token_infos"
}

func (b *cw20TokenInfoProcessor) UpsertStatement() string {
	return cw20TokenInfoTable.Upsert()
}

func (b *cw20TokenInfoProcessor) InsertStatement() string {
	return cw20TokenInfoTable.Insert()
}

func (b *cw20TokenInfoProcessor) DeleteStatement() string {
	return cw20TokenInfoTable.Delete()
}

func (b *cw20TokenInfoProcessor) SDKModuleName() tracelistener.SDKModuleName {
	return tracelistener.CW20
}

func (b *cw20TokenInfoProcessor) FlushCache() []tracelistener.WritebackOp {
	b.m.Lock()
	defer b.m.Unlock()

	if len(b.heightCache) == 0 {
		return nil
	}

	l := make([]models.DatabaseEntrier, 0, len(b.heightCache))

	for _, v := range b.heightCache {
		l = append(l, v)
	}

	b.heightCache = make(map[cw20TokenInfoCacheEntry]models.CW20TokenInfoRow)

	return []tracelistener.WritebackOp{
		{
			Type: tracelistener.Write,
			Data: l,
		},
	}
}

func (b *cw20TokenInfoProcessor) OwnsKey(key []byte) bool {
	return datamarshaler.IsCW20TokenInfoKey(key)
}

func (b *cw20TokenInfoProcessor) Process(data tracelistener.TraceOperation) error {
	b.m.Lock()
	defer b.m.Unlock()

	row, err := datamarshaler.NewDataMarshaler(b.l).CW20TokenInfo(data)
	if err != nil {
		return err
	}
	b.heightCache[cw20TokenInfoCacheEntry{
		contractAddress: row.ContractAddress,
	}] = row
	return nil
}
