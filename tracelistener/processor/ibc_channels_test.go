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

func TestIbcChannelsProcessOwnsKey(t *testing.T) {
	i := ibcChannelsProcessor{}

	tests := []struct {
		name        string
		prefix      []byte
		key         string
		expectedErr bool
	}{
		{
			"Correct prefix- no error",
			[]byte(datamarshaler.IBCChannelKey),
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

type testChannel struct {
	State            int32
	Ordering         int32
	CounterPortID    string
	CounterChannelID string
	Hop              string
}

func TestIbcChannelsProcess(t *testing.T) {
	i := ibcChannelsProcessor{}

	DataProcessor, _ := New(zap.NewNop().Sugar(), &config.Config{})
	gp := DataProcessor.(*Processor)
	require.NotNil(t, gp)

	tests := []struct {
		name        string
		channel     testChannel
		newMessage  tracelistener.TraceOperation
		expectedErr bool
		expectedLen int
	}{
		{
			"Ibc channel - no error",
			testChannel{
				State:            4,
				Ordering:         1,
				CounterPortID:    "some",
				CounterChannelID: "channelIdtest",
				Hop:              "connectionhopID",
			},
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
				Key:       []byte("cosmos/ports/x/channels/ibc"),
			},
			false,
			1,
		},
		{
			"Cannot parse channel path - error",
			testChannel{
				State:            4,
				Ordering:         1,
				CounterPortID:    "some",
				CounterChannelID: "channelIdtest",
				Hop:              "connectionhopID",
			},
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
				Key:       []byte("cosmos/x/channels/ibc"),
			},
			true,
			0,
		},
		{
			"Invalid counterparty port ID - error",
			testChannel{
				State:            4,
				Ordering:         1,
				CounterPortID:    "",
				CounterChannelID: "channelIdtest",
				Hop:              "connectionhopID",
			},
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
				Key:       []byte("cosmos/ports/x/channels/ibc"),
			},
			true,
			0,
		},
		{
			"Invalid connection hop ID - error",
			testChannel{
				State:            4,
				Ordering:         1,
				CounterPortID:    "some",
				CounterChannelID: "channelIdtest",
				Hop:              "",
			},
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
				Key:       []byte("cosmos/ports/x/channels/ibc"),
			},
			true,
			0,
		},
		{
			"Invalid channel state - error",
			testChannel{
				State:            0,
				Ordering:         1,
				CounterPortID:    "some",
				CounterChannelID: "channelIdtest",
				Hop:              "connectionhopID",
			},
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
				Key:       []byte("cosmos/ports/x/channels/ibc"),
			},
			true,
			0,
		},
		{
			"Invalid channel ordering - error",
			testChannel{
				State:            4,
				Ordering:         0,
				CounterPortID:    "some",
				CounterChannelID: "channelIdtest",
				Hop:              "connectionhopID",
			},
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
				Key:       []byte("cosmos/ports/x/channels/ibc"),
			},
			true,
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i.channelsCache = map[channelCacheEntry]models.IBCChannelRow{}
			i.l = zap.NewNop().Sugar()

			tt.newMessage.Value = datamarshaler.NewTestDataMarshaler().IBCChannel(
				tt.channel.State,
				tt.channel.Ordering,
				tt.channel.CounterPortID,
				tt.channel.CounterChannelID,
				tt.channel.Hop,
			)

			err := i.Process(tt.newMessage)
			if tt.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// check cache length
			require.Len(t, i.channelsCache, tt.expectedLen)

			// if channelcache not empty then check the data
			for k := range i.channelsCache {
				row := i.channelsCache[channelCacheEntry{channelID: k.channelID, portID: k.portID}]
				require.NotNil(t, row)

				state := row.State
				require.Equal(t, int32(tt.channel.State), state)

				return
			}
		})
	}
}

func TestIbcChannelFlushCache(t *testing.T) {
	i := ibcChannelsProcessor{}

	tests := []struct {
		name        string
		row         models.IBCChannelRow
		isNil       bool
		expectedNil bool
	}{
		{
			"Non empty data - No error",
			models.IBCChannelRow{
				ChannelID:        "channelId",
				Port:             "portId",
				CounterChannelID: "CounterChannelID",
			},
			false,
			false,
		},
		{
			"Empty data - error",
			models.IBCChannelRow{},
			true,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i.channelsCache = map[channelCacheEntry]models.IBCChannelRow{}

			if !tt.isNil {
				i.channelsCache[channelCacheEntry{
					channelID: tt.row.ChannelID,
					portID:    tt.row.Port,
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
