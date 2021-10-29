package gaia_processor

import (
	"bytes"
	"encoding/hex"
	"fmt"

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
	// How's a length-prefixed data.Key is made you ask?
	// 0x02<length prefix><address bytes>
	//
	// AddressFromBalancesStore requires the key data without the store prefix
	// so we simply reslice data.Key to get rid of it.
	//
	// If data.Operation == "delete", the trace that's been observed has a different data.Key:
	// 0x02<length prefix><address bytes><denom>
	//
	// This different schema is used when the balance associated to <denom> is being set to zero.
	// So, to obtain this denom one must subslice rawAddress to the length of <address bytes> + 1
	// to bypass the length prefix byte.
	rawAddress := data.Key[1:]
	addrBytes, err := types.AddressFromBalancesStore(rawAddress)
	if err != nil {
		return fmt.Errorf("cannot parse address from balance store key, %w", err)
	}

	hAddr := hex.EncodeToString(addrBytes)

	coins := sdk.Coin{
		Amount: sdk.NewInt(0),
	}

	if err := p.cdc.Unmarshal(data.Value, &coins); err != nil {
		return err
	}

	// Since SDK 0.44.x x/bank now deletes keys from store when the balance is 0
	// (picture someone who sends all their balance to another address).
	// To work around this issue, we don't return when coin is invalid when data.Operation is "delete",
	// and we set balance == 0 instead.
	if !coins.IsValid() {
		if data.Operation == tracelistener.DeleteOp.String() {
			// rawAddress still contains the lenght prefix, so we have to jump it by
			// reading 1 byte after len(addrBytes)
			denom := rawAddress[len(addrBytes)+1:]
			coins.Denom = string(denom)
		} else {
			return nil
		}
	}

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
