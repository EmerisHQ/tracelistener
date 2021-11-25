package gaia_processor

import (
	"testing"

	"github.com/stretchr/testify/require"
	liquiditytypes "github.com/tendermint/liquidity/x/liquidity/types"
	"go.uber.org/zap"

	models "github.com/allinbits/demeris-backend-models/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener/config"
)

func TestLiquidityPoolProcessOwnsKey(t *testing.T) {
	lp := liquidityPoolProcessor{}

	tests := []struct {
		name        string
		prefix      []byte
		key         string
		expectedErr bool
	}{
		{
			"Correct prefix- no error",
			liquiditytypes.PoolKeyPrefix,
			"key",
			false,
		},
		{
			"Incorrect prefix- error",
			[]byte{0x0},
			"key",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.expectedErr {
				require.False(t, lp.OwnsKey(append(tt.prefix, []byte(tt.key)...)))
			} else {
				require.True(t, lp.OwnsKey(append(tt.prefix, []byte(tt.key)...)))
			}
		})
	}
}

func TestLiquidityPoolProcess(t *testing.T) {
	l := liquidityPoolProcessor{}

	DataProcessor, err := New(zap.NewNop().Sugar(), &config.Config{})
	require.NoError(t, err)

	gp := DataProcessor.(*Processor)
	require.NotNil(t, gp)
	p.cdc = gp.cdc

	tests := []struct {
		name        string
		newMessage  tracelistener.TraceOperation
		lp          liquiditytypes.Pool
		expectedEr  bool
		expectedLen int
	}{
		{
			"Add liquidity pool details - no error",
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
			},
			liquiditytypes.Pool{
				Id:                    1,
				TypeId:                2,
				ReserveCoinDenoms:     []string{"atom", "akt"},
				ReserveAccountAddress: "cosmos1xrnner9s783446yz3hhshpr5fpz6wzcwkvwv5j",
			},
			false,
			1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l.poolsCache = map[uint64]models.PoolRow{}
			l.l = zap.NewNop().Sugar()

			value, err := p.cdc.MarshalBinaryBare(&tt.lp)
			require.NoError(t, err)
			tt.newMessage.Value = value

			err = l.Process(tt.newMessage)
			if tt.expectedEr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// check cache length
			require.Len(t, l.poolsCache, tt.expectedLen)

			// if poolcache not empty then check the data
			for k, _ := range l.poolsCache {
				row := l.poolsCache[k]
				require.NotNil(t, row)

				address := row.ReserveAccountAddress
				require.Equal(t, tt.lp.ReserveAccountAddress, address)

				return
			}
		})
	}
}

func TestLiquidityPoolFlushCache(t *testing.T) {
	l := liquidityPoolProcessor{}

	tests := []struct {
		name                  string
		poolID                uint64
		typeID                uint32
		poolCoinDenom         string
		reserveAccountAddress string
		isNil                 bool
		expectedNil           bool
	}{
		{
			"Non empty data - No error",
			2,
			1,
			"stake",
			"cosmos1xrnner9s783446yz3hhshpr5fpz6wzcwkvwv5j",
			false,
			false,
		},
		{
			"Empty data - error",
			0,
			0,
			"",
			"",
			true,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l.poolsCache = map[uint64]models.PoolRow{}

			if !tt.isNil {
				row := models.PoolRow{
					PoolID:                tt.poolID,
					TypeID:                tt.typeID,
					PoolCoinDenom:         tt.poolCoinDenom,
					ReserveAccountAddress: tt.reserveAccountAddress,
				}

				l.poolsCache[tt.poolID] = row
			}

			wop := l.FlushCache()
			if tt.expectedNil {
				require.Nil(t, wop)
			} else {
				require.NotNil(t, wop)
			}

			return
		})
	}
}
