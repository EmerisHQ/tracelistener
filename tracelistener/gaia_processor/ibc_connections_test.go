package gaia_processor

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/x/ibc/core/03-connection/types"
	ibcTypes "github.com/cosmos/cosmos-sdk/x/ibc/core/23-commitment/types"
	host "github.com/cosmos/cosmos-sdk/x/ibc/core/24-host"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	models "github.com/allinbits/demeris-backend-models/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener/config"
)

func TestIbcConnectionsProcess(t *testing.T) {
	i := ibcConnectionsProcessor{}

	// test ownkey prefix
	require.True(t, i.OwnsKey(append([]byte(host.KeyConnectionPrefix), []byte("key")...)))
	require.False(t, i.OwnsKey(append([]byte("0x0"), []byte("key")...)))

	DataProcessor, err := New(zap.NewNop().Sugar(), &config.Config{})
	require.NoError(t, err)

	gp := DataProcessor.(*Processor)
	require.NotNil(t, gp)
	p.cdc = gp.cdc

	tests := []struct {
		name        string
		newMessage  tracelistener.TraceOperation
		ce          types.ConnectionEnd
		expectedErr bool
		expectedLen int
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
			1,
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
			0,
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
			0,
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
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i.connectionsCache = map[connectionCacheEntry]models.IBCConnectionRow{}
			i.l = zap.NewNop().Sugar()

			value, err := p.cdc.MarshalBinaryBare(&tt.ce)
			require.NoError(t, err)
			tt.newMessage.Value = value

			err = i.Process(tt.newMessage)
			if tt.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// check cache length
			require.Len(t, i.connectionsCache, tt.expectedLen)

			// if connectioncache not empty then check the data
			for k, _ := range i.connectionsCache {
				row := i.connectionsCache[connectionCacheEntry{connectionID: k.connectionID, clientID: k.clientID}]
				require.NotNil(t, row)

				state := row.State
				require.Equal(t, tt.ce.State.String(), state)

				return
			}
		})
	}
}
