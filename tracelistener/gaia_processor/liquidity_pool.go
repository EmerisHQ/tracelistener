package gaia_processor

import (
	"bytes"
	"github.com/allinbits/demeris-backend/models"

	"github.com/allinbits/demeris-backend/tracelistener"
	liquiditytypes "github.com/tendermint/liquidity/x/liquidity/types"
	"go.uber.org/zap"
)

type liquidityPoolProcessor struct {
	l          *zap.SugaredLogger
	poolsCache map[uint64]models.PoolRow
}

func (*liquidityPoolProcessor) TableSchema() string {
	return createPoolsTable
}

func (b *liquidityPoolProcessor) ModuleName() string {
	return "liquidity_pools"
}

func (b *liquidityPoolProcessor) FlushCache() []tracelistener.WritebackOp {
	if len(b.poolsCache) == 0 {
		return nil
	}

	l := make([]models.DatabaseEntrier, 0, len(b.poolsCache))

	for _, c := range b.poolsCache {
		l = append(l, c)
	}

	b.poolsCache = map[uint64]models.PoolRow{}

	return []tracelistener.WritebackOp{
		{
			DatabaseExec: insertPool,
			Data:         l,
		},
	}
}

func (b *liquidityPoolProcessor) OwnsKey(key []byte) bool {
	return bytes.HasPrefix(key, liquiditytypes.PoolKeyPrefix)

}

func (b *liquidityPoolProcessor) Process(data tracelistener.TraceOperation) error {
	pool := liquiditytypes.Pool{}
	if err := p.cdc.UnmarshalBinaryBare(data.Value, &pool); err != nil {
		return err
	}

	b.poolsCache[pool.Id] = models.PoolRow{
		PoolID:                pool.Id,
		TypeID:                pool.TypeId,
		ReserveCoinDenoms:     pool.ReserveCoinDenoms,
		ReserveAccountAddress: pool.ReserveAccountAddress,
		PoolCoinDenom:         pool.PoolCoinDenom,
	}

	return nil
}
