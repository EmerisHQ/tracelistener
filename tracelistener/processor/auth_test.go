package processor

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	models "github.com/allinbits/demeris-backend-models/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener/config"
	"github.com/allinbits/tracelistener/tracelistener/processor/datamarshaler"
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
			datamarshaler.AuthKey,
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
				require.False(t, a.OwnsKey(append(tt.prefix, []byte(tt.key)...)))
			} else {
				require.True(t, a.OwnsKey(append(tt.prefix, []byte(tt.key)...)))
			}
		})
	}
}

type authAccount struct {
	Address       string
	AccountNumber uint64
	Sequence      uint64
}

func TestAuthProcess(t *testing.T) {
	a := authProcessor{}

	DataProcessor, err := New(zap.NewNop().Sugar(), &config.Config{})
	require.NoError(t, err)

	gp := DataProcessor.(*Processor)
	require.NotNil(t, gp)

	tests := []struct {
		name        string
		account     authAccount
		newMessage  tracelistener.TraceOperation
		expectedErr bool
		expectedLen int
	}{
		{
			"auth processor- no error",
			authAccount{
				Address:       "cosmos1xrnner9s783446yz3hhshpr5fpz6wzcwkvwv5j",
				AccountNumber: 12,
				Sequence:      11,
			},
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
				Key:       []byte("cosmos1xrnner9s783446"),
				Metadata: tracelistener.TraceMetadata{
					BlockHeight: 1,
				},
			},
			false,
			1,
		},
		{
			"invalid baseaccount address - error",
			authAccount{
				Address:       "",
				AccountNumber: 12,
				Sequence:      11,
			},
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
				Key:       []byte("cosmos1xrnner9s783446"),
				Metadata: tracelistener.TraceMetadata{
					BlockHeight: 1,
				},
			},
			true,
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			a.heightCache = map[authCacheEntry]models.AuthRow{}
			a.l = zap.NewNop().Sugar()

			tt.newMessage.Value = datamarshaler.NewTestDataMarshaler().Account(
				tt.account.AccountNumber, tt.account.Sequence, tt.account.Address,
			)

			err = a.Process(tt.newMessage)
			if tt.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Len(t, a.heightCache, tt.expectedLen)

			for k := range a.heightCache {
				row := a.heightCache[authCacheEntry{address: k.address, accNumber: k.accNumber}]
				require.NotNil(t, row)

				accountNumber := row.AccountNumber
				require.Equal(t, tt.account.AccountNumber, accountNumber)

				return
			}
		})
	}
}

func TestAuthFlushCache(t *testing.T) {
	a := authProcessor{}

	tests := []struct {
		name        string
		row         models.AuthRow
		isNil       bool
		expectedNil bool
	}{
		{
			"Non empty data- No error",
			models.AuthRow{
				Address:        "0A1E9FBE949F06AA6CABABF9262EF5C071DCA7E2",
				SequenceNumber: 1234,
				AccountNumber:  12,
			},
			false,
			false,
		},
		{
			"Empty data - error",
			models.AuthRow{},
			true,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a.heightCache = map[authCacheEntry]models.AuthRow{}

			if !tt.isNil {
				a.heightCache[authCacheEntry{
					address:   tt.row.Address,
					accNumber: tt.row.AccountNumber,
				}] = tt.row
			}

			wop := a.FlushCache()
			if tt.expectedNil {
				require.Nil(t, wop)
			} else {
				require.NotNil(t, wop)
			}
		})
	}
}
