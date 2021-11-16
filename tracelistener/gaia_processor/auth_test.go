package gaia_processor

import (
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	gaia "github.com/cosmos/gaia/v4/app"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	models "github.com/allinbits/demeris-backend-models/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener/config"
)

func TestAuthProcess(t *testing.T) {
	a := authProcessor{}

	DataProcessor, _ := New(zap.NewNop().Sugar(), &config.Config{})

	gp := DataProcessor.(*Processor)
	require.NotNil(t, gp)
	p.cdc = gp.cdc

	tests := []struct {
		name       string
		account    authtypes.BaseAccount
		newMessage tracelistener.TraceOperation
		wantErr    bool
	}{
		{
			"auth - no error",
			authtypes.BaseAccount{
				Address:       "cosmos1xrnner9s783446yz3hhshpr5fpz6wzcwkvwv5j",
				AccountNumber: 12,
				Sequence:      11,
			},
			tracelistener.TraceOperation{
				Operation:   string(tracelistener.WriteOp),
				Key:         []byte("cosmos1xrnner9s783446"),
				BlockHeight: 1,
			},
			false,
		},
		{
			"invalid baseaccount address - error",
			authtypes.BaseAccount{
				Address:       "",
				AccountNumber: 12,
				Sequence:      11,
			},
			tracelistener.TraceOperation{
				Operation:   string(tracelistener.WriteOp),
				Key:         []byte("cosmos1xrnner9s783446"),
				BlockHeight: 1,
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			a.heightCache = map[authCacheEntry]models.AuthRow{}
			a.l = zap.NewNop().Sugar()

			cdc, _ := gaia.MakeCodecs()

			delValue, _ := cdc.MarshalInterface(&tt.account)
			tt.newMessage.Value = delValue

			err := a.Process(tt.newMessage)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
