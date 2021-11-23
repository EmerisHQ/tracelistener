package gaia_processor

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/x/auth/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	models "github.com/allinbits/demeris-backend-models/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener/config"
)

func TestAuthOwnsKey(t *testing.T) {
	a := authProcessor{}

	tests := []struct {
		name        string
		prefix      []byte
		key         string
		expectedErr bool
	}{
		{
			"Correct prefix- no error",
			types.AddressStoreKeyPrefix,
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
				require.False(t, a.OwnsKey(append(tt.prefix, []byte(tt.key)...)))
			} else {
				require.True(t, a.OwnsKey(append(tt.prefix, []byte(tt.key)...)))
			}
		})
	}
}

func TestAuthProcess(t *testing.T) {
	a := authProcessor{}

	DataProcessor, err := New(zap.NewNop().Sugar(), &config.Config{})
	require.NoError(t, err)

	gp := DataProcessor.(*Processor)
	require.NotNil(t, gp)
	p.cdc = gp.cdc

	tests := []struct {
		name        string
		account     authtypes.BaseAccount
		newMessage  tracelistener.TraceOperation
		expectedErr bool
		expectedLen int
	}{
		{
			"auth processor- no error",
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
			1,
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
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			a.heightCache = map[authCacheEntry]models.AuthRow{}
			a.l = zap.NewNop().Sugar()

			delValue, err := p.cdc.MarshalInterface(&tt.account)
			require.NoError(t, err)
			tt.newMessage.Value = delValue

			err = a.Process(tt.newMessage)
			if tt.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Len(t, a.heightCache, tt.expectedLen)

			for k, _ := range a.heightCache {
				row := a.heightCache[authCacheEntry{address: k.address, accNumber: k.accNumber}]
				require.NotNil(t, row)

				accountNumber := row.AccountNumber
				require.Equal(t, tt.account.AccountNumber, accountNumber)
				return
			}

		})
	}
}
