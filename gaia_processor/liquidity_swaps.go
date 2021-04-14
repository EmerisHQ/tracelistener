package gaia_processor

import (
	"bytes"

	"github.com/allinbits/tracelistener"
	liquiditytypes "github.com/tendermint/liquidity/x/liquidity/types"
	"go.uber.org/zap"
)

type swapWritebackPacket struct {
	tracelistener.BasicDatabaseEntry

	MsgHeight            int64  `db:"msg_height"`
	MsgIndex             uint64 `db:"msg_index"`
	Executed             bool   `db:"executed"`
	Succeeded            bool   `db:"succeeded"`
	ExpiryHeight         int64  `db:"expiry_height"`
	ExchangedOfferCoin   string `db:"exchanged_offer_coin"`
	RemainingOfferCoin   string `db:"remaining_offer_coin"`
	ReservedOfferCoinFee string `db:"reserved_offer_coin_fee"`
	PoolCoinDenom        string `db:"pool_coin_denom"`
	RequesterAddress     string `db:"requester_address"`
	PoolID               uint64 `db:"pool_id"`
	OfferCoin            string `db:"offer_coin"`
	OrderPrice           string `db:"order_price"`
}

func (bwp swapWritebackPacket) WithChainName(cn string) tracelistener.DatabaseEntrier {
	bwp.ChainName = cn
	return bwp
}

type liquiditySwapsProcessor struct {
	l          *zap.SugaredLogger
	swapsCache map[uint64]swapWritebackPacket
}

func (*liquiditySwapsProcessor) TableSchema() string {
	return createSwapsTable
}

func (b *liquiditySwapsProcessor) ModuleName() string {
	return "liquidity_swaps"
}

func (b *liquiditySwapsProcessor) FlushCache() tracelistener.WritebackOp {
	if len(b.swapsCache) == 0 {
		return tracelistener.WritebackOp{}
	}

	l := make([]tracelistener.DatabaseEntrier, 0, len(b.swapsCache))

	for _, c := range b.swapsCache {
		l = append(l, c)
	}

	b.swapsCache = map[uint64]swapWritebackPacket{}

	return tracelistener.WritebackOp{
		DatabaseExec: insertSwap,
		Data:         l,
	}
}

func (b *liquiditySwapsProcessor) OwnsKey(key []byte) bool {
	return bytes.HasPrefix(key, liquiditytypes.PoolKeyPrefix)

}

func (b *liquiditySwapsProcessor) Process(data tracelistener.TraceOperation) error {
	swap := liquiditytypes.SwapMsgState{}
	if err := p.cdc.UnmarshalBinaryBare(data.Value, &swap); err != nil {
		return err
	}

	b.l.Debugw("new SwapMsgState", "content", swap.String())

	wbObj := swapWritebackPacket{
		BasicDatabaseEntry:   tracelistener.BasicDatabaseEntry{},
		MsgHeight:            swap.MsgHeight,
		MsgIndex:             swap.MsgIndex,
		Executed:             swap.Executed,
		Succeeded:            swap.Succeeded,
		ExpiryHeight:         swap.OrderExpiryHeight,
		ExchangedOfferCoin:   swap.ExchangedOfferCoin.String(),
		RemainingOfferCoin:   swap.RemainingOfferCoin.String(),
		ReservedOfferCoinFee: swap.ReservedOfferCoinFee.String(),
	}

	batch := swap.Msg

	if batch != nil {
		wbObj.PoolCoinDenom = batch.DemandCoinDenom
		wbObj.RequesterAddress = batch.SwapRequesterAddress
		wbObj.PoolID = batch.PoolId
		wbObj.OfferCoin = batch.OfferCoin.String()
		wbObj.OrderPrice = batch.OrderPrice.String()
	}

	b.swapsCache[swap.MsgIndex] = wbObj

	return nil
}
