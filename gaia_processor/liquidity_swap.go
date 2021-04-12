package gaia_processor

import (
	"bytes"

	"github.com/allinbits/tracelistener"
	liquiditytypes "github.com/tendermint/liquidity/x/liquidity/types"
	"go.uber.org/zap"
)

type swapWritebackPacket struct {
	tracelistener.BasicDatabaseEntry
}

func (bwp swapWritebackPacket) WithChainName(cn string) tracelistener.DatabaseEntrier {
	bwp.ChainName = cn
	return bwp
}

type liquiditySwapsProcessor struct {
	l          *zap.SugaredLogger
	swapsCache map[uint64]swapWritebackPacket
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

// TODO: finish swap processor
func (b *liquiditySwapsProcessor) Process(data tracelistener.TraceOperation) error {
	swap := liquiditytypes.SwapMsgState{}
	if err := p.cdc.UnmarshalBinaryBare(data.Value, &swap); err != nil {
		return err
	}

	return nil
}
