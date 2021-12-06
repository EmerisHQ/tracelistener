package gaia_processor

import (
	"testing"

	liquiditytypes "github.com/gravity-devs/liquidity/x/liquidity/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	models "github.com/allinbits/demeris-backend-models/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener/config"
)

func TestLiquiditySwapaProcessOwnsKey(t *testing.T) {
	ls := liquiditySwapsProcessor{}

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
				require.False(t, ls.OwnsKey(append(tt.prefix, []byte(tt.key)...)))
			} else {
				require.True(t, ls.OwnsKey(append(tt.prefix, []byte(tt.key)...)))
			}
		})
	}
}

func TestLiquiditySwapProcess(t *testing.T) {
	l := liquiditySwapsProcessor{}

	DataProcessor, err := New(zap.NewNop().Sugar(), &config.Config{})
	require.NoError(t, err)

	gp := DataProcessor.(*Processor)
	require.NotNil(t, gp)
	p.cdc = gp.cdc

	tests := []struct {
		name        string
		newMessage  tracelistener.TraceOperation
		ls          liquiditytypes.SwapMsgState
		expectedErr bool
		expectedLen int
	}{
		{
			"Liquidity swaps - no error",
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
			},
			liquiditytypes.SwapMsgState{
				MsgHeight: 120,
				MsgIndex:  1,
				Executed:  false,
				Msg: &liquiditytypes.MsgSwapWithinBatch{
					PoolId:          2,
					SwapTypeId:      1,
					DemandCoinDenom: "stake",
				},
			},
			false,
			1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l.swapsCache = map[uint64]models.SwapRow{}
			l.l = zap.NewNop().Sugar()

			value, err := p.cdc.MarshalBinaryBare(&tt.ls)
			require.NoError(t, err)
			tt.newMessage.Value = value

			err = l.Process(tt.newMessage)
			if tt.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// check cache length
			require.Len(t, l.swapsCache, tt.expectedLen)

			// if swapascache not empty then check the data
			for k := range l.swapsCache {
				row := l.swapsCache[k]
				require.NotNil(t, row)

				executed := row.Executed
				require.Equal(t, tt.ls.Executed, executed)

				return
			}
		})
	}
}

func TestLiquidityPoolSwapsFlushCache(t *testing.T) {
	l := liquiditySwapsProcessor{}

	tests := []struct {
		name             string
		msgHeight        int64
		poolID           uint64
		poolCoinDenom    string
		requesterAddress string
		isNil            bool
		expectedNil      bool
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
			l.swapsCache = map[uint64]models.SwapRow{}

			if !tt.isNil {
				row := models.SwapRow{
					PoolID:           tt.poolID,
					MsgHeight:        tt.msgHeight,
					PoolCoinDenom:    tt.poolCoinDenom,
					RequesterAddress: tt.requesterAddress,
				}

				l.swapsCache[tt.poolID] = row
			}

			wop := l.FlushCache()
			if tt.expectedNil {
				require.Nil(t, wop)
			} else {
				require.NotNil(t, wop)
			}
		})
	}
}
