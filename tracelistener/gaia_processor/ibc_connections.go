package gaia_processor

import (
	"bytes"
	"fmt"
	"strings"

	models "github.com/allinbits/demeris-backend-models/tracelistener"

	"github.com/cosmos/ibc-go/modules/core/03-connection/types"

	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"go.uber.org/zap"

	"github.com/allinbits/tracelistener/tracelistener"
)

type connectionCacheEntry struct {
	connectionID string
	clientID     string
}

var ibcObservedKeys = [][]byte{
	[]byte(host.KeyConnectionPrefix),
}

type ibcConnectionsProcessor struct {
	l                *zap.SugaredLogger
	connectionsCache map[connectionCacheEntry]models.IBCConnectionRow
}

func (*ibcConnectionsProcessor) TableSchema() string {
	return createConnectionsTable
}

func (b *ibcConnectionsProcessor) ModuleName() string {
	return "ibc_connections"
}

func (b *ibcConnectionsProcessor) FlushCache() []tracelistener.WritebackOp {
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
	keyFields := strings.FieldsFunc(string(data.Key), func(r rune) bool {
		return r == '/'
	})

	b.l.Debugw("ibc store key", "fields", keyFields, "raw key", string(data.Key))

	// IBC keys are mostly strings
	switch len(keyFields) {
	case 2:
		if keyFields[0] == host.KeyConnectionPrefix { // this is a ConnectionEnd
			ce := types.ConnectionEnd{}
			if err := p.cdc.Unmarshal(data.Value, &ce); err != nil {
				return fmt.Errorf("cannot unmarshal connection end, %w", err)
			}

			if err := ce.ValidateBasic(); err != nil {
				b.l.Debugw("found non-compliant connection end", "connection end", ce, "error", err)
				return fmt.Errorf("connection end validation failed, %w", err)
			}

			b.l.Debugw("connection end", "data", ce)

			b.connectionsCache[connectionCacheEntry{
				connectionID: keyFields[1],
				clientID:     ce.ClientId,
			}] = models.IBCConnectionRow{
				ConnectionID:        keyFields[1],
				ClientID:            ce.ClientId,
				State:               ce.State.String(),
				CounterConnectionID: ce.Counterparty.ConnectionId,
				CounterClientID:     ce.Counterparty.ClientId,
			}
		}
	}

	return nil
}
