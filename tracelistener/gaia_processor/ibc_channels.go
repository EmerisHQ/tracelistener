package gaia_processor

import (
	"bytes"
	"fmt"

	"github.com/allinbits/tracelistener/models"

	"github.com/cosmos/ibc-go/modules/core/04-channel/types"

	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"go.uber.org/zap"

	"github.com/allinbits/tracelistener/tracelistener"
)

type channelCacheEntry struct {
	channelID string
	portID    string
}

type ibcChannelsProcessor struct {
	l             *zap.SugaredLogger
	channelsCache map[channelCacheEntry]models.IBCChannelRow
}

func (*ibcChannelsProcessor) TableSchema() string {
	return createChannelsTable
}

func (b *ibcChannelsProcessor) ModuleName() string {
	return "ibc_channels"
}

func (b *ibcChannelsProcessor) FlushCache() []tracelistener.WritebackOp {
	if len(b.channelsCache) == 0 {
		return nil
	}

	l := make([]models.DatabaseEntrier, 0, len(b.channelsCache))

	for _, c := range b.channelsCache {
		l = append(l, c)
	}

	b.channelsCache = map[channelCacheEntry]models.IBCChannelRow{}

	return []tracelistener.WritebackOp{
		{
			DatabaseExec: insertChannel,
			Data:         l,
		},
	}
}

func (b *ibcChannelsProcessor) OwnsKey(key []byte) bool {
	return bytes.HasPrefix(key, []byte(host.KeyChannelEndPrefix))
}

func (b *ibcChannelsProcessor) Process(data tracelistener.TraceOperation) error {
	b.l.Debugw("ibc channel key", "key", string(data.Key), "raw value", string(data.Value))
	var result types.Channel
	if err := p.cdc.Unmarshal(data.Value, &result); err != nil {
		return err
	}

	if err := result.ValidateBasic(); err != nil {
		b.l.Debugw("found non-compliant channel", "channel", result, "error", err)
		return fmt.Errorf("cannot validate ibc channel, %w", err)
	}

	if result.Ordering != types.UNORDERED {
		return nil
	}

	b.l.Debugw("ibc channel data", "result", result)

	portID, channelID, err := host.ParseChannelPath(string(data.Key))
	if err != nil {
		return err
	}

	b.channelsCache[channelCacheEntry{
		channelID: channelID,
		portID:    portID,
	}] = models.IBCChannelRow{
		ChannelID:        channelID,
		CounterChannelID: result.Counterparty.ChannelId,
		Hops:             result.GetConnectionHops(),
		Port:             portID,
		State:            int32(result.State),
	}

	return nil
}
