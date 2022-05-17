package processor

import (
	"bytes"
	"sync"

	models "github.com/emerishq/demeris-backend-models/tracelistener"

	"go.uber.org/zap"

	"github.com/emerishq/tracelistener/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener/processor/datamarshaler"
	"github.com/emerishq/tracelistener/tracelistener/tables"
)

var channelsTable = tables.NewChannelsTable("tracelistener.channels")

type channelCacheEntry struct {
	channelID string
	portID    string
}

type ibcChannelsProcessor struct {
	l             *zap.SugaredLogger
	channelsCache map[channelCacheEntry]models.IBCChannelRow
	m             sync.Mutex
}

func (*ibcChannelsProcessor) Migrations() []string {
	return []string{channelsTable.CreateTable()}
}

func (b *ibcChannelsProcessor) ModuleName() string {
	return "ibc_channels"
}

func (b *ibcChannelsProcessor) SDKModuleName() tracelistener.SDKModuleName {
	return tracelistener.IBC
}

func (b *ibcChannelsProcessor) UpsertStatement() string {
	return channelsTable.Upsert()
}

func (b *ibcChannelsProcessor) InsertStatement() string {
	return channelsTable.Insert()
}

func (b *ibcChannelsProcessor) DeleteStatement() string {
	panic("ibc channel processor never deletes")
}

func (b *ibcChannelsProcessor) FlushCache() []tracelistener.WritebackOp {
	b.m.Lock()
	defer b.m.Unlock()

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
			Type: tracelistener.Write,
			Data: l,
		},
	}
}

func (b *ibcChannelsProcessor) OwnsKey(key []byte) bool {
	return bytes.HasPrefix(key, []byte(datamarshaler.IBCChannelKey))
}

func (b *ibcChannelsProcessor) Process(data tracelistener.TraceOperation) error {
	b.m.Lock()
	defer b.m.Unlock()

	res, err := datamarshaler.NewDataMarshaler(b.l).IBCChannels(data)
	if err != nil {
		return err
	}

	b.channelsCache[channelCacheEntry{
		channelID: res.ChannelID,
		portID:    res.Port,
	}] = res

	return nil
}
