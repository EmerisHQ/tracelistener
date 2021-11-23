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
			for k, _ := range l.swapsCache {
				row := l.swapsCache[k]
				require.NotNil(t, row)

				executed := row.Executed
				require.Equal(t, tt.ls.Executed, executed)

				return
			}
		})
	}
}
