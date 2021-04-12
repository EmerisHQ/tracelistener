package gaia_processor

import (
	"bytes"
	"strings"

	"github.com/cosmos/cosmos-sdk/x/ibc/core/03-connection/types"

	host "github.com/cosmos/cosmos-sdk/x/ibc/core/24-host"
	"go.uber.org/zap"

	"github.com/allinbits/tracelistener"
)

type connectionWritebackPacket struct {
	tracelistener.BasicDatabaseEntry

	ConnectionID        string `db:"connection_id" json:"connection_id"`
	ClientID            string `db:"client_id" json:"client_id"`
	State               string `db:"state" json:"state"`
	CounterConnectionID string `db:"counter_connection_id" json:"counter_connection_id"`
	CounterClientID     string `db:"counter_client_id" json:"counter_client_id"`
}

func (c connectionWritebackPacket) WithChainName(cn string) tracelistener.DatabaseEntrier {
	c.ChainName = cn
	return c
}

type connectionCacheEntry struct {
	connectionID string
	clientID     string
}

var ibcObservedKeys = [][]byte{
	[]byte(host.KeyConnectionPrefix),
}

type ibcProcessor struct {
	l                *zap.SugaredLogger
	connectionsCache map[connectionCacheEntry]connectionWritebackPacket
}

func (b *ibcProcessor) ModuleName() string {
	return "ibc"
}

func (b *ibcProcessor) FlushCache() tracelistener.WritebackOp {
	if len(b.connectionsCache) == 0 {
		return tracelistener.WritebackOp{}
	}

	l := make([]tracelistener.DatabaseEntrier, 0, len(b.connectionsCache))

	for _, c := range b.connectionsCache {
		l = append(l, c)
	}

	b.connectionsCache = map[connectionCacheEntry]connectionWritebackPacket{}

	return tracelistener.WritebackOp{
		DatabaseExec: insertConnection,
		Data:         l,
	}
}

func (b *ibcProcessor) OwnsKey(key []byte) bool {
	for _, k := range ibcObservedKeys {
		if bytes.HasPrefix(key, k) {
			return true
		}
	}

	return false
}

func (b *ibcProcessor) Process(data tracelistener.TraceOperation) error {
	keyFields := strings.FieldsFunc(string(data.Key), func(r rune) bool {
		return r == '/'
	})

	b.l.Debugw("ibc store key", "fields", keyFields, "raw key", string(data.Key))

	// IBC keys are mostly strings
	switch len(keyFields) {
	case 2:
		if keyFields[0] == host.KeyConnectionPrefix { // this is a ConnectionEnd
			ce := types.ConnectionEnd{}
			p.cdc.MustUnmarshalBinaryBare(data.Value, &ce)
			b.l.Debugw("connection end", "data", ce)

			b.connectionsCache[connectionCacheEntry{
				connectionID: keyFields[1],
				clientID:     ce.ClientId,
			}] = connectionWritebackPacket{
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
