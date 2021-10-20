package gaia_processor

import (
	"bytes"
	"encoding/hex"

	models "github.com/allinbits/demeris-backend-models/tracelistener"

	"go.uber.org/zap"

	"github.com/allinbits/tracelistener/tracelistener"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/x/bank/types"
)

type bankCacheEntry struct {
	address string
	denom   string
}

type bankProcessor struct {
	l           *zap.SugaredLogger
	heightCache map[bankCacheEntry]models.BalanceRow
}

func (*bankProcessor) TableSchema() string {
	return createBalancesTable
}

func (b *bankProcessor) ModuleName() string {
	return "bank"
}

func (b *bankProcessor) FlushCache() []tracelistener.WritebackOp {
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
			DatabaseExec: insertBalance,
			Data:         l,
		},
	}
}

func (b *bankProcessor) OwnsKey(key []byte) bool {
	return bytes.HasPrefix(key, types.BalancesPrefix)
}

func (b *bankProcessor) Process(data tracelistener.TraceOperation) error {
	addrBytes := data.Key
	pLen := len(types.BalancesPrefix)
	addr := addrBytes[pLen : pLen+20]

	coins := sdk.Coin{
		Amount: sdk.NewInt(0),
	}

	if err := p.cdc.UnmarshalBinaryBare(data.Value, &coins); err != nil {
		return err
	}

	if !coins.IsValid() {
		return nil
	}

	hAddr := hex.EncodeToString(addr)
	b.l.Debugw("new bank store write",
		"operation", data.Operation,
		"address", hAddr,
		"new_balance", coins.String(),
		"height", data.BlockHeight,
		"txHash", data.TxHash,
	)

	b.heightCache[bankCacheEntry{
		address: hAddr,
		denom:   coins.Denom,
	}] = models.BalanceRow{
		Address:     hAddr,
		Amount:      coins.String(),
		Denom:       coins.Denom,
		BlockHeight: data.BlockHeight,
	}

	return nil
}
