package moduleprocessor

import (
	"bytes"
	"context"

	sdkserviceclient "github.com/allinbits/sdk-service-meta/gen/grpc/sdk_utilities/client"
	sdkutilities "github.com/allinbits/sdk-service-meta/gen/sdk_utilities"
	"google.golang.org/grpc"

	tracelistener2 "github.com/allinbits/tracelistener"

	"github.com/allinbits/tracelistener/models"

	host "github.com/cosmos/cosmos-sdk/x/ibc/core/24-host"
	"go.uber.org/zap"
)

type channelCacheEntry struct {
	channelID string
	portID    string
}

type ibcChannelsProcessor struct {
	l             *zap.SugaredLogger
	grpcConn      *grpc.ClientConn
	channelsCache map[channelCacheEntry]models.IBCChannelRow
}

func (*ibcChannelsProcessor) TableSchema() string {
	return createChannelsTable
}

func (b *ibcChannelsProcessor) ModuleName() string {
	return "ibc_channels"
}

func (b *ibcChannelsProcessor) FlushCache() []tracelistener2.WritebackOp {
	if len(b.channelsCache) == 0 {
		return nil
	}

	l := make([]models.DatabaseEntrier, 0, len(b.channelsCache))

	for _, c := range b.channelsCache {
		l = append(l, c)
	}

	b.channelsCache = map[channelCacheEntry]models.IBCChannelRow{}

	return []tracelistener2.WritebackOp{
		{
			DatabaseExec: insertChannel,
			Data:         l,
		},
	}
}

func (b *ibcChannelsProcessor) OwnsKey(key []byte) bool {
	return bytes.HasPrefix(key, []byte(host.KeyChannelEndPrefix))
}

func (b *ibcChannelsProcessor) Process(data tracelistener2.TraceOperation) error {
	client := sdkserviceclient.NewClient(b.grpcConn)

	cc := sdkutilities.Client{
		IbcChannelEndpoint: client.IbcChannel(),
	}

	payload := sdkutilities.TracePayload{
		Key:           data.Key,
		Value:         data.Value,
		OperationType: &data.Operation,
	}

	res, err := cc.IbcChannel(context.Background(), &sdkutilities.IbcChannelPayload{
		Payload: []*sdkutilities.TracePayload{
			&payload,
		},
	})

	if err != nil {
		return err
	}

	for _, r := range res {
		b.channelsCache[channelCacheEntry{
			channelID: *r.ChannelID,
			portID:    *r.Port,
		}] = models.IBCChannelRow{
			ChannelID:        *r.ChannelID,
			CounterChannelID: *r.CounterChannelID,
			Hops:             r.Hops,
			Port:             *r.Port,
			State:            *r.State,
		}
	}

	return nil
}
