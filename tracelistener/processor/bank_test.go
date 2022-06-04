package processor

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	models "github.com/emerishq/demeris-backend-models/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener/config"
	"github.com/emerishq/tracelistener/tracelistener/processor/datamarshaler"
)

func TestBankProcessorOwnsKey(t *testing.T) {
	d := bankProcessor{}
	tests := []struct {
		name         string
		key          []byte
		expectedOwns bool
	}{
		{
			name:         "Correct prefix- no error",
			key:          append(datamarshaler.BankKey, []byte{1, 1, 'a', 't', 'o', 'm'}...),
			expectedOwns: true,
		},
		{
			name:         "Incorrect prefix- error",
			key:          []byte{0x0},
			expectedOwns: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			owns := d.OwnsKey(tt.key)

			require.Equal(t, tt.expectedOwns, owns)
		})
	}
}

type testCoin struct {
	Denom  string
	Amount int64
}

func TestBankProcess(t *testing.T) {
	b := bankProcessor{}

	DataProcessor, err := New(zap.NewNop().Sugar(), &config.Config{})
	require.NoError(t, err)

	gp := DataProcessor.(*Processor)
	require.NotNil(t, gp)

	tests := []struct {
		name        string
		coin        testCoin
		newMessage  tracelistener.TraceOperation
		expectedErr bool
		expectedLen int
	}{
		{
			"No error of bank process",
			testCoin{
				Denom:  "stake",
				Amount: 500,
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

			tt.newMessage.Key = datamarshaler.NewTestDataMarshaler().BankAddress(string(tt.newMessage.Key))
			tt.newMessage.Value = datamarshaler.NewTestDataMarshaler().Coin(
				tt.coin.Denom,
				tt.coin.Amount,
			)

			err = b.Process(tt.newMessage)
			if tt.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Len(t, b.heightCache, tt.expectedLen)

			for k := range b.heightCache {
				row := b.heightCache[bankCacheEntry{address: k.address, denom: k.denom}]
				require.NotNil(t, row)

				denom := row.Denom
				require.Equal(t, tt.coin.Denom, denom)

				require.Equal(t, tt.newMessage.BlockHeight, row.Height)
			}
		})
	}
}

func TestBankFlushCache(t *testing.T) {
	b := bankProcessor{}

	tests := []struct {
		name        string
		row         models.BalanceRow
		isNil       bool
		expectedNil bool
	}{
		{
			"Non empty data - No error",
			models.BalanceRow{
				Address: "0A1E9FBE949F06AA6CABABF9262EF5C071DCA7E2",
				Denom:   "stake",
				Amount:  "100stake",
			},
			false,
			false,
		},
		{
			"Empty data - error",
			models.BalanceRow{},
			true,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b.heightCache = map[bankCacheEntry]models.BalanceRow{}

			if !tt.isNil {
				b.heightCache[bankCacheEntry{
					address: tt.row.Address,
					denom:   tt.row.Denom,
				}] = tt.row
			}

			wop := b.FlushCache()
			if tt.expectedNil {
				require.Nil(t, wop)
			} else {
				require.NotNil(t, wop)
			}
		})
	}
}
