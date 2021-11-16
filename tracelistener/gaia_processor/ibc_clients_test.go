package gaia_processor

import (
	"testing"

	ics23 "github.com/confio/ics23/go"
	"github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	clientTypes "github.com/cosmos/cosmos-sdk/x/ibc/light-clients/07-tendermint/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	models "github.com/allinbits/demeris-backend-models/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener/config"
)

func TestIbcClientProcess(t *testing.T) {
	b := ibcClientsProcessor{}

	DataProcessor, _ := New(zap.NewNop().Sugar(), &config.Config{})

	gp := DataProcessor.(*Processor)
	require.NotNil(t, gp)
	p.cdc = gp.cdc

	tests := []struct {
		name       string
		newMessage tracelistener.TraceOperation
		v          clientTypes.ClientState
		wantErr    bool
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b.clientsCache = map[clientCacheEntry]models.IBCClientStateRow{}
			b.l = zap.NewNop().Sugar()

			value, _ := p.cdc.MarshalInterface(&tt.v)
			tt.newMessage.Value = value
			err := b.Process(tt.newMessage)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
