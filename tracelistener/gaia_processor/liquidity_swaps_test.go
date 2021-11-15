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

func TestLiquiditySwapProcess(t *testing.T) {
	b := liquiditySwapsProcessor{}

	DataProcessor, _ := New(zap.NewNop().Sugar(), &config.Config{})

	gp := DataProcessor.(*Processor)
	require.NotNil(t, gp)
	p.cdc = gp.cdc

	tests := []struct {
		name       string
		newMessage tracelistener.TraceOperation
		ls         liquiditytypes.SwapMsgState
		wantErr    bool
	}{
		{
			"Liquidity swaos - no error",
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
			},
			liquiditytypes.SwapMsgState{
				MsgHeight: 120,
				MsgIndex:  1,
				Executed:  false,
				// Id:                    1,
				// TypeId:                2,
				// ReserveCoinDenoms:     []string{"atom", "akt"},
				// ReserveAccountAddress: "cosmos1xrnner9s783446yz3hhshpr5fpz6wzcwkvwv5j",
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b.swapsCache = map[uint64]models.SwapRow{}
			b.l = zap.NewNop().Sugar()

			delValue, _ := p.cdc.MarshalBinaryBare(&tt.ls)
			tt.newMessage.Value = delValue

			err := b.Process(tt.newMessage)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
