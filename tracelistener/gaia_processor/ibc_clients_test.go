package gaia_processor

import (
	"testing"

	ics23 "github.com/confio/ics23/go"
	"github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	host "github.com/cosmos/cosmos-sdk/x/ibc/core/24-host"
	clientTypes "github.com/cosmos/cosmos-sdk/x/ibc/light-clients/07-tendermint/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	models "github.com/allinbits/demeris-backend-models/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener/config"
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
			[]byte(host.KeyClientState),
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
	p.cdc = gp.cdc

	tests := []struct {
		name        string
		newMessage  tracelistener.TraceOperation
		client      clientTypes.ClientState
		expectedErr bool
		expectedLen int
	}{
		{
			"Ibc connection - no error",
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
				Key:       []byte("some/text"),
			},
			clientTypes.ClientState{
				ChainId: "cosmos",
				TrustLevel: clientTypes.Fraction{
					Numerator:   1,
					Denominator: 3,
				},
				TrustingPeriod:  1,
				UnbondingPeriod: 2,
				MaxClockDrift:   2,
				ProofSpecs: []*ics23.ProofSpec{
					{
						LeafSpec: &ics23.LeafOp{
							Hash:   1,
							Length: 1,
						},
					},
				},
				FrozenHeight: types.Height{
					RevisionNumber: 100,
					RevisionHeight: 120,
				},
				LatestHeight: types.NewHeight(100, 102),
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
			clientTypes.ClientState{
				ChainId: "cosmos",
				TrustLevel: clientTypes.Fraction{
					Numerator:   1,
					Denominator: 3,
				},
				TrustingPeriod:  3,
				UnbondingPeriod: 2,
				MaxClockDrift:   2,
				ProofSpecs: []*ics23.ProofSpec{
					{
						LeafSpec: &ics23.LeafOp{
							Hash:   1,
							Length: 1,
						},
					},
				},
				FrozenHeight: types.Height{
					RevisionNumber: 100,
					RevisionHeight: 120,
				},
				LatestHeight: types.NewHeight(100, 102),
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
			clientTypes.ClientState{
				ChainId: "cosmos",
				TrustLevel: clientTypes.Fraction{
					Numerator:   1,
					Denominator: 3,
				},
				TrustingPeriod:  1,
				UnbondingPeriod: 2,
				MaxClockDrift:   0,
				ProofSpecs: []*ics23.ProofSpec{
					{
						LeafSpec: &ics23.LeafOp{
							Hash:   1,
							Length: 1,
						},
					},
				},
				FrozenHeight: types.Height{
					RevisionNumber: 100,
					RevisionHeight: 120,
				},
				LatestHeight: types.NewHeight(100, 102),
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
			clientTypes.ClientState{
				ChainId: "cosmos",
				TrustLevel: clientTypes.Fraction{
					Numerator:   1,
					Denominator: 3,
				},
				TrustingPeriod:  3,
				UnbondingPeriod: 1,
				MaxClockDrift:   2,
				ProofSpecs: []*ics23.ProofSpec{
					{
						LeafSpec: &ics23.LeafOp{
							Hash:   1,
							Length: 1,
						},
					},
				},
				FrozenHeight: types.Height{
					RevisionNumber: 100,
					RevisionHeight: 120,
				},
				LatestHeight: types.NewHeight(100, 102),
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
			clientTypes.ClientState{
				ChainId: "cosmos",
				TrustLevel: clientTypes.Fraction{
					Numerator:   1,
					Denominator: 3,
				},
				TrustingPeriod:  1,
				UnbondingPeriod: 2,
				MaxClockDrift:   2,
				ProofSpecs:      []*ics23.ProofSpec{},
				FrozenHeight: types.Height{
					RevisionNumber: 100,
					RevisionHeight: 120,
				},
				LatestHeight: types.NewHeight(100, 102),
			},
			true,
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i.clientsCache = map[clientCacheEntry]models.IBCClientStateRow{}
			i.l = zap.NewNop().Sugar()

			value, err := p.cdc.MarshalInterface(&tt.client)
			require.NoError(t, err)
			tt.newMessage.Value = value

			err = i.Process(tt.newMessage)
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
				require.Equal(t, tt.client.ChainId, chainID)

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
			models.IBCClientStateRow{
				ChainID:      "",
				ClientID:     "",
				LatestHeight: 0,
			},
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
				}] = models.IBCClientStateRow{
					ChainID:      tt.row.ChainID,
					ClientID:     tt.row.ClientID,
					LatestHeight: tt.row.LatestHeight,
				}
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
