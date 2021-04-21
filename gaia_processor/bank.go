package gaia_processor

import (
	"bytes"
	"encoding/hex"

	"go.uber.org/zap"

	"github.com/allinbits/tracelistener"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/x/bank/types"
)

type balanceWritebackPacket struct {
	tracelistener.BasicDatabaseEntry

	Address     string `db:"address" json:"address"`
	Amount      string `db:"amount" json:"amount"`
	Denom       string `db:"denom" json:"denom"`
	BlockHeight uint64 `db:"height" json:"block_height"`
}

func (b balanceWritebackPacket) WithChainName(cn string) tracelistener.DatabaseEntrier {
	b.ChainName = cn
	return b
}

type bankCacheEntry struct {
	address string
	denom   string
}

type bankProcessor struct {
	l           *zap.SugaredLogger
	heightCache map[bankCacheEntry]balanceWritebackPacket
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

	l := make([]tracelistener.DatabaseEntrier, 0, len(b.heightCache))

	for _, v := range b.heightCache {
		l = append(l, v)
	}

	b.heightCache = map[bankCacheEntry]balanceWritebackPacket{}

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

	coins := sdk.Coin{}

	if err := p.cdc.UnmarshalBinaryBare(data.Value, &coins); err != nil {
		return err
	}

	if coins.Amount.IsNil() || coins.IsZero() || !coins.IsValid() {
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
	}] = balanceWritebackPacket{
		Address:     hAddr,
		Amount:      coins.String(),
		Denom:       coins.Denom,
		BlockHeight: data.BlockHeight,
	}

	return nil
}
