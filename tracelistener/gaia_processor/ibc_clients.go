package gaia_processor

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/x/ibc/core/exported"

	tmIBCTypes "github.com/cosmos/cosmos-sdk/x/ibc/light-clients/07-tendermint/types"

	models "github.com/allinbits/demeris-backend-models/tracelistener"

	host "github.com/cosmos/cosmos-sdk/x/ibc/core/24-host"
	"go.uber.org/zap"

	"github.com/allinbits/tracelistener/tracelistener"
)

type clientCacheEntry struct {
	chainID  string
	clientID string
}

type ibcClientsProcessor struct {
	l            *zap.SugaredLogger
	clientsCache map[clientCacheEntry]models.IBCClientStateRow
}

func (*ibcClientsProcessor) TableSchema() string {
	return createClientsTable
}

func (b *ibcClientsProcessor) ModuleName() string {
	return "ibc_clients"
}

func (b *ibcClientsProcessor) FlushCache() []tracelistener.WritebackOp {
	if len(b.clientsCache) == 0 {
		return nil
	}

	l := make([]models.DatabaseEntrier, 0, len(b.clientsCache))

	for _, c := range b.clientsCache {
		l = append(l, c)
	}

	b.clientsCache = map[clientCacheEntry]models.IBCClientStateRow{}

	return []tracelistener.WritebackOp{
		{
			DatabaseExec: insertClient,
			Data:         l,
		},
	}
}

func (b *ibcClientsProcessor) OwnsKey(key []byte) bool {
	return bytes.Contains(key, []byte(host.KeyClientState))
}

func (b *ibcClientsProcessor) Process(data tracelistener.TraceOperation) error {
	b.l.Debugw("ibc client key", "key", string(data.Key), "raw value", string(data.Value))
	var result exported.ClientState
	var dest *tmIBCTypes.ClientState
	if err := p.cdc.UnmarshalInterface(data.Value, &result); err != nil {
		return err
	}

	if res, ok := result.(*tmIBCTypes.ClientState); !ok {
		return nil
	} else {
		dest = res
	}

	if err := result.Validate(); err != nil {
		b.l.Debugw("found non-compliant ibc connection", "connection", dest, "error", err)
		return fmt.Errorf("cannot validate ibc connection, %w", err)
	}

	keySplit := strings.Split(string(data.Key), "/")
	clientID := keySplit[1]

	b.clientsCache[clientCacheEntry{
		chainID:  dest.ChainId,
		clientID: clientID,
	}] = models.IBCClientStateRow{
		ChainID:        dest.ChainId,
		ClientID:       clientID,
		LatestHeight:   dest.LatestHeight.RevisionHeight,
		TrustingPeriod: int64(dest.TrustingPeriod),
	}

	return nil
}
