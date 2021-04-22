package gaia_processor

import (
	"bytes"

	"github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"

	host "github.com/cosmos/cosmos-sdk/x/ibc/core/24-host"
	"go.uber.org/zap"

	"github.com/allinbits/tracelistener"
)

type channelWritebackPacket struct {
	tracelistener.BasicDatabaseEntry

	ChannelID string   `db:"channel_id" json:"channel_id"`
	Hops      []string `db:"hops" json:"hops"`
	Port      string   `db:"port" json:"port"`
	State     int32    `db:"state" json:"state"`
}

func (c channelWritebackPacket) WithChainName(cn string) tracelistener.DatabaseEntrier {
	c.ChainName = cn
	return c
}

type channelCacheEntry struct {
	channelID string
	portID    string
}

type ibcChannelsProcessor struct {
	l             *zap.SugaredLogger
	channelsCache map[channelCacheEntry]channelWritebackPacket
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

	l := make([]tracelistener.DatabaseEntrier, 0, len(b.channelsCache))

	for _, c := range b.channelsCache {
		l = append(l, c)
	}

	b.channelsCache = map[channelCacheEntry]channelWritebackPacket{}

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
	if err := p.cdc.UnmarshalBinaryBare(data.Value, &result); err != nil {
		return err
	}

	portID, channelID, err := host.ParseChannelPath(string(data.Key))
	if err != nil {
		return err
	}

	b.channelsCache[channelCacheEntry{
		channelID: channelID,
		portID:    portID,
	}] = channelWritebackPacket{
		ChannelID: channelID,
		Hops:      result.GetConnectionHops(),
		Port:      portID,
		State:     int32(result.State),
	}

	return nil
}
