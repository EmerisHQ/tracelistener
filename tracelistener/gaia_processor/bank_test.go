package gaia_processor

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	models "github.com/allinbits/demeris-backend-models/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener/config"
)

func TestBankProcessorOwnsKey(t *testing.T) {
	d := bankProcessor{}

	tests := []struct {
		name        string
		prefix      []byte
		key         string
		expectedErr bool
	}{
		{
			"Correct prefix- no error",
			types.BalancesPrefix,
			"key",
			false,
		},
		{
			"Incorrect prefix- error",
			[]byte("0x0"),
			"key",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.expectedErr {
				require.False(t, d.OwnsKey(append(tt.prefix, []byte(tt.key)...)))
			} else {
				require.True(t, d.OwnsKey(append(tt.prefix, []byte(tt.key)...)))
			}
		})
	}
}

func TestBankProcess(t *testing.T) {
	b := bankProcessor{}

	DataProcessor, err := New(zap.NewNop().Sugar(), &config.Config{})
	require.NoError(t, err)

	gp := DataProcessor.(*Processor)
	require.NotNil(t, gp)
	p.cdc = gp.cdc

	tests := []struct {
		name        string
		coin        sdk.Coin
		newMessage  tracelistener.TraceOperation
		expectedErr bool
		expectedLen int
	}{
		{
			"No error of bank process",
			sdk.Coin{
				Denom:  "stake",
				Amount: sdk.NewInt(500),
			},
			tracelistener.TraceOperation{
				Operation:   string(tracelistener.WriteOp),
				Key:         []byte("cosmos1xrnner9s783446yz3hhshpr5fpz6wzcwkvwv5j"),
				BlockHeight: 101,
			},
			false,
			1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b.heightCache = map[bankCacheEntry]models.BalanceRow{}
			b.l = zap.NewNop().Sugar()

			value, err := p.cdc.MarshalBinaryBare(&tt.coin)
			require.NoError(t, err)
			tt.newMessage.Value = value

			err = b.Process(tt.newMessage)
			if tt.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Len(t, b.heightCache, tt.expectedLen)

			for k, _ := range b.heightCache {
				row := b.heightCache[bankCacheEntry{address: k.address, denom: k.denom}]
				require.NotNil(t, row)

				denom := row.Denom
				require.Equal(t, tt.coin.Denom, denom)
				return
			}
		})
	}
}
