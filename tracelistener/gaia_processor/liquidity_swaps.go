package gaia_processor

import (
	"bytes"

	models "github.com/allinbits/demeris-backend-models/tracelistener"

	"github.com/allinbits/tracelistener/tracelistener"
	liquiditytypes "github.com/gravity-devs/liquidity/x/liquidity/types"
	"go.uber.org/zap"
)

type liquiditySwapsProcessor struct {
	l          *zap.SugaredLogger
	swapsCache map[uint64]models.SwapRow
}

func (*liquiditySwapsProcessor) TableSchema() string {
	return createSwapsTable
}

func (b *liquiditySwapsProcessor) ModuleName() string {
	return "liquidity_swaps"
}

func (b *liquiditySwapsProcessor) FlushCache() []tracelistener.WritebackOp {
	if len(b.swapsCache) == 0 {
		return nil
	}

	l := make([]models.DatabaseEntrier, 0, len(b.swapsCache))

	for _, c := range b.swapsCache {
		l = append(l, c)
	}

	b.swapsCache = map[uint64]models.SwapRow{}

	return []tracelistener.WritebackOp{
		{
			DatabaseExec: insertSwap,
			Data:         l,
		},
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

	wbObj := models.SwapRow{
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
