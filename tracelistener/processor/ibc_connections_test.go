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

func TestIbcConnProcessOwnsKey(t *testing.T) {
	i := ibcConnectionsProcessor{}

	tests := []struct {
		name        string
		prefix      []byte
		key         string
		expectedErr bool
	}{
		{
			"Correct prefix- no error",
			[]byte(datamarshaler.IBCConnectionsKey),
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

func TestIbcConnectionsProcess(t *testing.T) {
	i := ibcConnectionsProcessor{}

	DataProcessor, err := New(zap.NewNop().Sugar(), &config.Config{})
	require.NoError(t, err)

	gp := DataProcessor.(*Processor)
	require.NotNil(t, gp)

	tests := []struct {
		name        string
		newMessage  tracelistener.TraceOperation
		ce          datamarshaler.TestConnection
		expectedErr bool
		expectedLen int
	}{
		{
			"Ibc connection - no error",
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
				Key:       []byte("connections/2"),
			},
			datamarshaler.TestConnection{
				ClientId:          "clientidtest",
				VersionIdentifier: "ibc",
				State:             1,
				CountClientID:     "counterpartyclientid",
				CountConnectionID: "counterpartyconnid",
				CountPrefix:       "prefix",
				DelayPeriod:       12,
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
			datamarshaler.TestConnection{
				VersionIdentifier: "ibc",
				State:             1,
				CountClientID:     "counterpartyclientid",
				CountConnectionID: "counterpartyconnid",
				CountPrefix:       "prefix",
				DelayPeriod:       12,
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
			datamarshaler.TestConnection{
				ClientId:          "clientidtest",
				VersionIdentifier: "ibc",
				State:             1,
				CountClientID:     "id",
				CountConnectionID: "conn",
				CountPrefix:       "prefix",
				DelayPeriod:       2,
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
			datamarshaler.TestConnection{
				ClientId:          "clientidtest",
				VersionIdentifier: "ibc",
				State:             1,
				CountClientID:     "counterpartyclientid",
				CountConnectionID: "counterpartyconnid",
				CountPrefix:       "",
				DelayPeriod:       12,
			},
			true,
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i.connectionsCache = map[connectionCacheEntry]models.IBCConnectionRow{}
			i.l = zap.NewNop().Sugar()

			tt.newMessage.Value = datamarshaler.NewTestDataMarshaler().IBCConnection(tt.ce)

			err = i.Process(tt.newMessage)
			if tt.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// check cache length
			require.Len(t, i.connectionsCache, tt.expectedLen)

			// if connectioncache not empty then check the data
			for k := range i.connectionsCache {
				row := i.connectionsCache[connectionCacheEntry{connectionID: k.connectionID, clientID: k.clientID}]
				require.NotNil(t, row)

				state := row.State
				require.Equal(t, datamarshaler.NewTestDataMarshaler().MapConnectionState(tt.ce.State), state)

				return
			}
		})
	}
}

func TestIbcConnectionsFlushCache(t *testing.T) {
	i := ibcConnectionsProcessor{}

	tests := []struct {
		name        string
		row         models.IBCConnectionRow
		isNil       bool
		expectedNil bool
	}{
		{
			"Non empty data - No error",
			models.IBCConnectionRow{
				ConnectionID: "connectionID",
				ClientID:     "clientID",
			},
			false,
			false,
		},
		{
			"Empty data - error",
			models.IBCConnectionRow{},
			true,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i.connectionsCache = map[connectionCacheEntry]models.IBCConnectionRow{}

			if !tt.isNil {
				i.connectionsCache[connectionCacheEntry{
					connectionID: tt.row.ConnectionID,
					clientID:     tt.row.ClientID,
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
