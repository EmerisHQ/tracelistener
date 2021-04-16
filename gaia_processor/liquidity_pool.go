package gaia_processor

import (
	"bytes"

	"github.com/allinbits/tracelistener"
	liquiditytypes "github.com/tendermint/liquidity/x/liquidity/types"
	"go.uber.org/zap"
)

type poolWritebackPacket struct {
	tracelistener.BasicDatabaseEntry

	PoolID                uint64   `db:"pool_id"`
	TypeID                uint32   `db:"type_id"`
	ReserveCoinDenoms     []string `db:"reserve_coin_denoms"`
	ReserveAccountAddress string   `db:"reserve_account_address"`
	PoolCoinDenom         string   `db:"pool_coin_denom"`
}

func (bwp poolWritebackPacket) WithChainName(cn string) tracelistener.DatabaseEntrier {
	bwp.ChainName = cn
	return bwp
}

type liquidityPoolProcessor struct {
	l          *zap.SugaredLogger
	poolsCache map[uint64]poolWritebackPacket
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

	l := make([]tracelistener.DatabaseEntrier, 0, len(b.poolsCache))

	for _, c := range b.poolsCache {
		l = append(l, c)
	}

	b.poolsCache = map[uint64]poolWritebackPacket{}

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

	b.poolsCache[pool.Id] = poolWritebackPacket{
		PoolID:                pool.Id,
		TypeID:                pool.TypeId,
		ReserveCoinDenoms:     pool.ReserveCoinDenoms,
		ReserveAccountAddress: pool.ReserveAccountAddress,
		PoolCoinDenom:         pool.PoolCoinDenom,
	}

	return nil
}
