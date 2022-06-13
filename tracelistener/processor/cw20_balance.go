package processor

import (
	"sync"

	"go.uber.org/zap"

	models "github.com/emerishq/demeris-backend-models/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener/processor/datamarshaler"
	"github.com/emerishq/tracelistener/tracelistener/tables"
)

var cw20BalanceTable = tables.NewCw20BalancesTable("tracelistener.cw20_balances")

type cw20BalanceCacheEntry struct {
	address         string
	contractAddress string
}

type cw20BalanceProcessor struct {
	l           *zap.SugaredLogger
	heightCache map[cw20BalanceCacheEntry]models.CW20BalanceRow
	m           sync.Mutex
}

func (*cw20BalanceProcessor) Migrations() []string {
	return []string{cw20BalanceTable.CreateTable()}
}

func (b *cw20BalanceProcessor) ModuleName() string {
	return "cw20_balances"
}

func (b *cw20BalanceProcessor) UpsertStatement() string {
	return cw20BalanceTable.Upsert()
}

func (b *cw20BalanceProcessor) InsertStatement() string {
	return cw20BalanceTable.Insert()
}

func (b *cw20BalanceProcessor) DeleteStatement() string {
	return cw20BalanceTable.Delete()
}

func (b *cw20BalanceProcessor) SDKModuleName() tracelistener.SDKModuleName {
	return tracelistener.CW20
}

func (b *cw20BalanceProcessor) FlushCache() []tracelistener.WritebackOp {
	b.m.Lock()
	defer b.m.Unlock()

	if len(b.heightCache) == 0 {
		return nil
	}

	l := make([]models.DatabaseEntrier, 0, len(b.heightCache))

	for _, v := range b.heightCache {
		l = append(l, v)
	}

	b.heightCache = make(map[cw20BalanceCacheEntry]models.CW20BalanceRow)

	return []tracelistener.WritebackOp{
		{
			Type: tracelistener.Write,
			Data: l,
		},
	}
}

func (b *cw20BalanceProcessor) OwnsKey(key []byte) bool {
	return datamarshaler.IsCW20BalanceKey(key)
}

func (b *cw20BalanceProcessor) Process(data tracelistener.TraceOperation) error {
	b.m.Lock()
	defer b.m.Unlock()

	row, err := datamarshaler.NewDataMarshaler(b.l).CW20Balance(data)
	if err != nil {
		return err
	}
	key := cw20BalanceCacheEntry{
		contractAddress: row.ContractAddress,
		address:         row.Address,
	}
	b.heightCache[key] = row
	return nil
}
