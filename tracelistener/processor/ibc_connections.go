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

var connectionsTable = tables.NewConnectionsTable("tracelistener.connections")

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

func (*ibcConnectionsProcessor) Migrations() []string {
	return []string{connectionsTable.CreateTable()}
}

func (b *ibcConnectionsProcessor) ModuleName() string {
	return "ibc_connections"
}

func (b *ibcConnectionsProcessor) SDKModuleName() tracelistener.SDKModuleName {
	return tracelistener.IBC
}

func (b *ibcConnectionsProcessor) UpsertStatement() string {
	return connectionsTable.Upsert()
}

func (b *ibcConnectionsProcessor) InsertStatement() string {
	return connectionsTable.Insert()
}

func (b *ibcConnectionsProcessor) DeleteStatement() string {
	panic("ibc connections processor never deletes")
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
			Type: tracelistener.Write,
			Data: l,
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
