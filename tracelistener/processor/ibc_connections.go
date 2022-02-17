package processor

import (
	"bytes"
	"sync"

	models "github.com/allinbits/demeris-backend-models/tracelistener"

	"go.uber.org/zap"

	"github.com/allinbits/tracelistener/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener/processor/datamarshaler"
)

type connectionCacheEntry struct {
	connectionID string
	clientID     string
}

var ibcObservedKeys = [][]byte{
	[]byte(datamarshaler.IBCConnectionsKey),
}

type ibcConnectionsProcessor struct {
	l                *zap.SugaredLogger
	connectionsCache map[connectionCacheEntry]models.IBCConnectionRow
	m                sync.Mutex
}

func (*ibcConnectionsProcessor) TableSchema() string {
	return createConnectionsTable
}

func (b *ibcConnectionsProcessor) ModuleName() string {
	return "ibc_connections"
}

func (b *ibcConnectionsProcessor) SDKModuleName() string {
	return "ibc"
}

func (b *ibcConnectionsProcessor) FlushCache() []tracelistener.WritebackOp {
	b.m.Lock()
	defer b.m.Unlock()

	if len(b.connectionsCache) == 0 {
		return nil
	}

	l := make([]models.DatabaseEntrier, 0, len(b.connectionsCache))

	for _, c := range b.connectionsCache {
		l = append(l, c)
	}

	b.connectionsCache = map[connectionCacheEntry]models.IBCConnectionRow{}

	return []tracelistener.WritebackOp{
		{
			DatabaseExec: insertConnection,
			Data:         l,
		},
	}
}

func (b *ibcConnectionsProcessor) OwnsKey(key []byte) bool {
	for _, k := range ibcObservedKeys {
		if bytes.HasPrefix(key, k) {
			return true
		}
	}

	return false
}

func (b *ibcConnectionsProcessor) Process(data tracelistener.TraceOperation) error {
	b.m.Lock()
	defer b.m.Unlock()

	res, err := datamarshaler.NewDataMarshaler(b.l).IBCConnections(data)
	if err != nil {
		return err
	}

	b.connectionsCache[connectionCacheEntry{
		connectionID: res.ConnectionID,
		clientID:     res.ClientID,
	}] = res

	return nil
}
