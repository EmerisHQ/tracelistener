package processor

import (
	"bytes"
	"sync"

	"go.uber.org/zap"

	models "github.com/emerishq/demeris-backend-models/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener/processor/datamarshaler"
	"github.com/emerishq/tracelistener/tracelistener/tables"
)

var balancesTable = tables.NewBalancesTable("tracelistener.balances")

type bankCacheEntry struct {
	address string
	denom   string
}

type bankProcessor struct {
	l           *zap.SugaredLogger
	heightCache map[bankCacheEntry]models.BalanceRow
	m           sync.Mutex
}

func (*bankProcessor) Migrations() []string {
	if useSQLGen {
		return []string{balancesTable.CreateTable()}
	} else {
		return []string{createBalancesTable, addHeightColumn(balanceTableOld), addDeleteHeightColumn(balanceTableOld)}
	}
}

func (b *bankProcessor) ModuleName() string {
	return "bank"
}

func (b *bankProcessor) UpsertStatement() string {
	if useSQLGen {
		return balancesTable.Upsert()
	} else {
		return upsertBalance
	}
}

func (b *bankProcessor) InsertStatement() string {
	if useSQLGen {
		return balancesTable.Insert()
	} else {
		return insertBalance
	}
}

func (b *bankProcessor) DeleteStatement() string {
	panic("bank processor never deletes")
}

func (b *bankProcessor) SDKModuleName() tracelistener.SDKModuleName {
	return tracelistener.Bank
}

func (b *bankProcessor) FlushCache() []tracelistener.WritebackOp {
	b.m.Lock()
	defer b.m.Unlock()

	if len(b.heightCache) == 0 {
		return nil
	}

	l := make([]models.DatabaseEntrier, 0, len(b.heightCache))

	for _, v := range b.heightCache {
		l = append(l, v)
	}

	b.heightCache = map[bankCacheEntry]models.BalanceRow{}

	return []tracelistener.WritebackOp{
		{
			Type: tracelistener.Write,
			Data: l,
		},
	}
}

func (b *bankProcessor) OwnsKey(key []byte) bool {
	return bytes.HasPrefix(key, datamarshaler.BankKey)
}

func (b *bankProcessor) Process(data tracelistener.TraceOperation) error {
	b.m.Lock()
	defer b.m.Unlock()

	res, err := datamarshaler.NewDataMarshaler(b.l).Bank(data)
	if err != nil {
		return err
	}

	b.heightCache[bankCacheEntry{
		address: res.Address,
		denom:   res.Denom,
	}] = res

	return nil
}
