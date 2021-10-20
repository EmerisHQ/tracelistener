package gaia_processor

import (
	"bytes"
	"fmt"

	"github.com/cosmos/cosmos-sdk/x/bank/exported"

	"github.com/allinbits/demeris-backend/models"

	"go.uber.org/zap"

	"github.com/allinbits/demeris-backend/tracelistener"

	"github.com/cosmos/cosmos-sdk/x/bank/types"
)

type supplyCacheEntry struct {
	denom string
}

type supplyProcessor struct {
	l           *zap.SugaredLogger
	heightCache map[supplyCacheEntry]models.SupplyRow
}

func (*supplyProcessor) TableSchema() string {
	return createSupplyTable
}

func (b *supplyProcessor) ModuleName() string {
	return "supply"
}

func (b *supplyProcessor) FlushCache() []tracelistener.WritebackOp {
	if len(b.heightCache) == 0 {
		return nil
	}

	l := make([]models.DatabaseEntrier, 0, len(b.heightCache))

	for _, v := range b.heightCache {
		l = append(l, v)
	}

	b.heightCache = map[supplyCacheEntry]models.SupplyRow{}

	return []tracelistener.WritebackOp{
		{
			DatabaseExec: insertSupply,
			Data:         l,
		},
	}
}

func (b *supplyProcessor) OwnsKey(key []byte) bool {
	return bytes.HasPrefix(key, types.SupplyKey)
}

func (b *supplyProcessor) Process(data tracelistener.TraceOperation) error {
	var evi exported.SupplyI

	if err := p.cdc.UnmarshalInterface(data.Value, &evi); err != nil {
		b.l.Debugw("cannot unmarshal into SupplyI", "error", err)
		return fmt.Errorf("cannot unmarshal supply data into object, %w", err)
	}

	if err := evi.ValidateBasic(); err != nil {
		b.l.Debugw("supply validatebasic failed", "error", err, "object", evi)
		return fmt.Errorf("supply validatebasic failed, %w", err)
	}

	for _, c := range evi.GetTotal() {
		b.heightCache[supplyCacheEntry{
			denom: c.GetDenom(),
		}] = models.SupplyRow{
			Amount: c.String(),
			Denom:  c.Denom,
		}
	}

	return nil
}
