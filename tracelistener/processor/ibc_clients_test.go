package processor

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	models "github.com/emerishq/demeris-backend-models/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener/config"
	"github.com/emerishq/tracelistener/tracelistener/processor/datamarshaler"
)

func TestIbcClientProcessOwnsKey(t *testing.T) {
	i := ibcClientsProcessor{}

	tests := []struct {
		name        string
		prefix      []byte
		key         string
		expectedErr bool
	}{
		{
			"Correct prefix- no error",
			[]byte(datamarshaler.IBCClientsKey),
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
				require.False(t, i.OwnsKey(append(tt.prefix, []byte(tt.key)...)))
			} else {
				require.True(t, i.OwnsKey(append(tt.prefix, []byte(tt.key)...)))
			}
		})
	}
}

func TestIbcClientProcess(t *testing.T) {
	i := ibcClientsProcessor{}

	DataProcessor, _ := New(zap.NewNop().Sugar(), &config.Config{})

	gp := DataProcessor.(*Processor)
	require.NotNil(t, gp)

	tests := []struct {
		name        string
		newMessage  tracelistener.TraceOperation
		client      datamarshaler.TestClientState
		expectedErr bool
		expectedLen int
	}{
		{
			"Ibc connection - no error",
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
				Key:       []byte("some/text"),
			},
			datamarshaler.TestClientState{
				ChainId: "cosmos-102",
				TrustLevel: datamarshaler.TestFraction{
					Numerator:   1,
					Denominator: 3,
				},
				TrustingPeriod:  1,
				UnbondingPeriod: 2,
				MaxClockDrift:   2,
				ProofSpecs: []datamarshaler.TestProofSpec{
					{
						Hash:   1,
						Length: 1,
					},
				},
				FrozenHeight: datamarshaler.TestHeight{
					Number: 100,
					Height: 120,
				},
				LatestHeight: datamarshaler.TestHeight{
					Number: 100,
					Height: 102,
				},
			},
			false,
			1,
		},
		{
			"Trusting period should be < unbonding period - error",
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
				Key:       []byte("connections/info"),
				Value:     []byte("some"),
			},
			datamarshaler.TestClientState{
				ChainId: "cosmos-102",
				TrustLevel: datamarshaler.TestFraction{
					Numerator:   1,
					Denominator: 3,
				},
				TrustingPeriod:  3,
				UnbondingPeriod: 2,
				MaxClockDrift:   2,
				ProofSpecs: []datamarshaler.TestProofSpec{
					{
						Hash:   1,
						Length: 1,
					},
				},
				FrozenHeight: datamarshaler.TestHeight{
					Number: 100,
					Height: 120,
				},
				LatestHeight: datamarshaler.TestHeight{
					Number: 100,
					Height: 102,
				},
			},
			true,
			0,
		},
		{
			"Max clock drift cannot be zero - error",
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
				Key:       []byte("some/text"),
				Value:     []byte("some"),
			},
			datamarshaler.TestClientState{
				ChainId: "cosmos-102",
				TrustLevel: datamarshaler.TestFraction{
					Numerator:   1,
					Denominator: 3,
				},
				TrustingPeriod:  1,
				UnbondingPeriod: 2,
				MaxClockDrift:   0,
				ProofSpecs: []datamarshaler.TestProofSpec{
					{
						Hash:   1,
						Length: 1,
					},
				},
				FrozenHeight: datamarshaler.TestHeight{
					Number: 100,
					Height: 120,
				},
				LatestHeight: datamarshaler.TestHeight{
					Number: 100,
					Height: 102,
				},
			},
			true,
			0,
		},
		{
			"TrustLevel must be within [1/3, 1] - error",
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
				Key:       []byte("some/text"),
				Value:     []byte("some"),
			},
			datamarshaler.TestClientState{
				ChainId: "cosmos-102",
				TrustLevel: datamarshaler.TestFraction{
					Numerator:   1,
					Denominator: 3,
				},
				TrustingPeriod:  3,
				UnbondingPeriod: 1,
				MaxClockDrift:   2,
				ProofSpecs: []datamarshaler.TestProofSpec{
					{
						Hash:   1,
						Length: 1,
					},
				},
				FrozenHeight: datamarshaler.TestHeight{
					Number: 100,
					Height: 120,
				},
				LatestHeight: datamarshaler.TestHeight{
					Number: 100,
					Height: 102,
				},
			},
			true,
			0,
		},
		{
			"Proof specs cannot be nil for tm client - error",
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
				Key:       []byte("some/text"),
				Value:     []byte("some"),
			},
			datamarshaler.TestClientState{
				ChainId: "cosmos",
				TrustLevel: datamarshaler.TestFraction{
					Numerator:   1,
					Denominator: 3,
				},
				TrustingPeriod:  1,
				UnbondingPeriod: 2,
				MaxClockDrift:   2,
				ProofSpecs:      nil,
				FrozenHeight: datamarshaler.TestHeight{
					Number: 100,
					Height: 120,
				},
				LatestHeight: datamarshaler.TestHeight{
					Number: 100,
					Height: 102,
				},
			},
			true,
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i.clientsCache = map[clientCacheEntry]models.IBCClientStateRow{}
			i.l = zap.NewNop().Sugar()

			tt.newMessage.Value = datamarshaler.NewTestDataMarshaler().IBCClient(tt.client)

			err := i.Process(tt.newMessage)
			if tt.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// check cache length
			require.Len(t, i.clientsCache, tt.expectedLen)

			// if clientcache not empty then check the data
			for k := range i.clientsCache {
				row := i.clientsCache[clientCacheEntry{chainID: k.chainID, clientID: k.clientID}]
				require.NotNil(t, row)

				chainID := row.ChainID
				require.True(t, strings.HasPrefix(tt.client.ChainId, chainID))

				return
			}
		})
	}
}

func TestIbcClientsFlushCache(t *testing.T) {
	i := ibcClientsProcessor{}

	tests := []struct {
		name        string
		row         models.IBCClientStateRow
		isNil       bool
		expectedNil bool
	}{
		{
			"Non empty data - No error",
			models.IBCClientStateRow{
				ChainID:      "cosmos",
				ClientID:     "clientID",
				LatestHeight: 4211,
			},
			false,
			false,
		},
		{
			"Empty data - error",
			models.IBCClientStateRow{},
			true,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i.clientsCache = map[clientCacheEntry]models.IBCClientStateRow{}

			if !tt.isNil {
				i.clientsCache[clientCacheEntry{
					chainID:  tt.row.ChainID,
					clientID: tt.row.ClientID,
				}] = tt.row
			}

			wop := i.FlushCache()
			if tt.expectedNil {
				require.Nil(t, wop)
			} else {
				require.NotNil(t, wop)
			}
		})
	}
}
