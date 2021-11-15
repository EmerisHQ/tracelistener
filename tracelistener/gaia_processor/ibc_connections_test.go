package gaia_processor

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/x/ibc/core/03-connection/types"
	ibcTypes "github.com/cosmos/cosmos-sdk/x/ibc/core/23-commitment/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	models "github.com/allinbits/demeris-backend-models/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener/config"
)

func TestIbcConnectionsProcess(t *testing.T) {
	b := ibcConnectionsProcessor{}

	DataProcessor, _ := New(zap.NewNop().Sugar(), &config.Config{})

	gp := DataProcessor.(*Processor)
	require.NotNil(t, gp)
	p.cdc = gp.cdc

	tests := []struct {
		name       string
		newMessage tracelistener.TraceOperation
		ce         types.ConnectionEnd
		wantErr    bool
	}{
		{
			"Ibc connection - no error",
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
				Key:       []byte("connections/2"),
			},
			types.ConnectionEnd{
				ClientId: "clientidtest",
				Versions: []*types.Version{
					{
						Identifier: "ibc",
					},
				},
				State: types.State(1),
				Counterparty: types.Counterparty{
					ClientId:     "counterpartyclientid",
					ConnectionId: "counterpartyconnid",
					Prefix: ibcTypes.MerklePrefix{
						KeyPrefix: []byte("prefix"),
					},
				},
				DelayPeriod: 12,
			},
			false,
		},
		{
			"Empty client id - error",
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
				Key:       []byte("connections/2"),
			},
			types.ConnectionEnd{
				ClientId: "",
				Versions: []*types.Version{
					{
						Identifier: "ibc",
					},
				},
				State: types.State(1),
				Counterparty: types.Counterparty{
					ClientId:     "counterpartyclientid",
					ConnectionId: "counterpartyconnid",
					Prefix: ibcTypes.MerklePrefix{
						KeyPrefix: []byte("prefix"),
					},
				},
				DelayPeriod: 2,
			},
			true,
		},
		{
			"Invalid length of counterparty client and connection id - error",
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
				Key:       []byte("connections/2"),
			},
			types.ConnectionEnd{
				ClientId: "clientidtest",
				Versions: []*types.Version{
					{
						Identifier: "ibc",
					},
				},
				State: types.State(1),
				Counterparty: types.Counterparty{
					ClientId:     "id",
					ConnectionId: "conn",
					Prefix: ibcTypes.MerklePrefix{
						KeyPrefix: []byte("prefix"),
					},
				},
				DelayPeriod: 2,
			},
			true,
		},
		{
			"Invalid counterparty prefix - error",
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
				Key:       []byte("connections/2"),
			},
			types.ConnectionEnd{
				ClientId: "clientidtest",
				Versions: []*types.Version{
					{
						Identifier: "ibc",
					},
				},
				State: types.State(1),
				Counterparty: types.Counterparty{
					ClientId:     "counterpartyclientid",
					ConnectionId: "counterpartyconnid",
					Prefix: ibcTypes.MerklePrefix{
						KeyPrefix: []byte(""),
					},
				},
				DelayPeriod: 2,
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b.connectionsCache = map[connectionCacheEntry]models.IBCConnectionRow{}
			b.l = zap.NewNop().Sugar()

			value, _ := p.cdc.MarshalBinaryBare(&tt.ce)
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
