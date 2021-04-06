package gaia_processor

import (
	"bytes"
	"encoding/hex"

	"github.com/allinbits/tracelistener"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/x/bank/types"
)

type balanceWritebackPacket struct {
	ID          uint64 `db:"id" json:"-"`
	Address     string `db:"address" json:"address"`
	Amount      uint64 `db:"amount" json:"amount"`
	Denom       string `db:"denom" json:"denom"`
	BlockHeight uint64 `db:"height" json:"block_height"`
}

type cacheEntry struct {
	address string
	denom   string
}

type bankProcessor struct {
	heightCache map[cacheEntry]balanceWritebackPacket
}

func (b *bankProcessor) ModuleName() string {
	return "bank"
}

func (b *bankProcessor) FlushCache() tracelistener.WritebackOp {
	if len(b.heightCache) == 0 {
		return tracelistener.WritebackOp{}
	}

	l := make([]interface{}, 0, len(b.heightCache))

	for _, v := range b.heightCache {
		l = append(l, v)
	}

	b.heightCache = map[cacheEntry]balanceWritebackPacket{}

	return tracelistener.WritebackOp{
		DatabaseExec: insertBalanceQuery,
		Data:         l,
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
	b.heightCache[cacheEntry{
		address: hAddr,
		denom:   coins.Denom,
	}] = balanceWritebackPacket{
		Address:     hAddr,
		Amount:      coins.Amount.Uint64(),
		Denom:       coins.Denom,
		BlockHeight: data.BlockHeight,
	}

	return nil
}
