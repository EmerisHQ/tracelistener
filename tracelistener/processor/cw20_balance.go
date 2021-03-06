package processor

import (
	"sync"

	"go.uber.org/zap"

	"github.com/emerishq/tracelistener/models"
	"github.com/emerishq/tracelistener/tracelistener"
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
	_, _, err := tracelistener.SplitCW20BalanceKey(key)
	return err == nil
}

func (b *cw20BalanceProcessor) Process(data tracelistener.TraceOperation) error {
	b.m.Lock()
	defer b.m.Unlock()

	contractAddr, holderAddr, err := tracelistener.SplitCW20BalanceKey(data.Key)
	if err != nil {
		return err
	}
	var (
		key = cw20BalanceCacheEntry{
			contractAddress: contractAddr,
			address:         holderAddr,
		}
		val = models.CW20BalanceRow{
			ContractAddress: contractAddr,
			Address:         holderAddr,
			// balance trace value is the amount.
			Amount: string(data.Value),
			TracelistenerDatabaseRow: models.TracelistenerDatabaseRow{
				Height: data.BlockHeight,
			},
		}
	)
	b.heightCache[key] = val
	return nil
}
