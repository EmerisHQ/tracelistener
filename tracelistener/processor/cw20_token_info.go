package processor

import (
	"encoding/json"
	"fmt"
	"sync"

	"go.uber.org/zap"

	models "github.com/emerishq/demeris-backend-models/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener"
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
	panic("cw20TokenInfo processor never deletes")
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
	_, err := tracelistener.SplitCW20TokenInfoKey(key)
	return err == nil
}

func (b *cw20TokenInfoProcessor) Process(data tracelistener.TraceOperation) error {
	b.m.Lock()
	defer b.m.Unlock()

	contractAddr, err := tracelistener.SplitCW20TokenInfoKey(data.Key)
	if err != nil {
		return err
	}
	var (
		key = cw20TokenInfoCacheEntry{
			contractAddress: contractAddr,
		}
		val = models.CW20TokenInfoRow{
			ContractAddress: contractAddr,
			TracelistenerDatabaseRow: models.TracelistenerDatabaseRow{
				Height: data.BlockHeight,
			},
		}
	)
	// token_info value is a json string that contains the name, the symbol,
	// the decimals and the total_supply of the token. To copy those values in
	// the CW20TokenInfoRow, we can simply json.Unmarshal the value to the struct.
	err = json.Unmarshal(data.Value, &val)
	if err != nil {
		return fmt.Errorf("unmarshal cw20 token_info value: %w", err)
	}
	b.heightCache[key] = val
	return nil
}
