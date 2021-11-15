package gaia_processor

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	models "github.com/allinbits/demeris-backend-models/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener/config"
)

func TestIbcChannelsProcess(t *testing.T) {
	u := ibcChannelsProcessor{}

	DataProcessor, _ := New(zap.NewNop().Sugar(), &config.Config{})
	gp := DataProcessor.(*Processor)
	require.NotNil(t, gp)
	p.cdc = gp.cdc

	tests := []struct {
		name       string
		channel    types.Channel
		newMessage tracelistener.TraceOperation
		wantErr    bool
	}{
		{
			"Ibc channel - no error",
			types.Channel{
				State:    4,
				Ordering: 1,
				Counterparty: types.Counterparty{
					PortId:    "some",
					ChannelId: "channelIdtest",
				},
				ConnectionHops: []string{"connectionhopID"},
			},
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
				Key:       []byte("cosmos/ports/x/channels/ibc"),
			},
			false,
		},
		{
			"Cannot parse channel path - error",
			types.Channel{
				State:    4,
				Ordering: 1,
				Counterparty: types.Counterparty{
					PortId:    "some",
					ChannelId: "channelIdtest",
				},
				ConnectionHops: []string{"connectionhopID"},
			},
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
				Key:       []byte("cosmos/x/channels/ibc"),
			},
			true,
		},
		{
			"Invalid counterparty port ID - error",
			types.Channel{
				State:    4,
				Ordering: 1,
				Counterparty: types.Counterparty{
					PortId:    "",
					ChannelId: "channelIdtest",
				},
				ConnectionHops: []string{"connectionhopID"},
			},
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
				Key:       []byte("cosmos/ports/x/channels/ibc"),
			},
			true,
		},
		{
			"Invalid connection hop ID - error",
			types.Channel{
				State:    4,
				Ordering: 1,
				Counterparty: types.Counterparty{
					PortId:    "some",
					ChannelId: "channelIdtest",
				},
				ConnectionHops: []string{""},
			},
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
				Key:       []byte("cosmos/ports/x/channels/ibc"),
			},
			true,
		},
		{
			"Invalid channel state - error",
			types.Channel{
				State:    0,
				Ordering: 1,
				Counterparty: types.Counterparty{
					PortId:    "some",
					ChannelId: "channelIdtest",
				},
				ConnectionHops: []string{"connectionhopID"},
			},
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
				Key:       []byte("cosmos/ports/x/channels/ibc"),
			},
			true,
		},
		{
			"Invalid channel ordering - error",
			types.Channel{
				State:    4,
				Ordering: 0,
				Counterparty: types.Counterparty{
					PortId:    "some",
					ChannelId: "channelIdtest",
				},
				ConnectionHops: []string{"connectionhopID"},
			},
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
				Key:       []byte("cosmos/ports/x/channels/ibc"),
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u.channelsCache = map[channelCacheEntry]models.IBCChannelRow{}
			u.l = zap.NewNop().Sugar()

			delValue, _ := p.cdc.MarshalBinaryBare(&tt.channel)
			tt.newMessage.Value = delValue

			err := u.Process(tt.newMessage)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
