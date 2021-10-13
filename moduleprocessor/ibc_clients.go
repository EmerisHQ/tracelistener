package moduleprocessor

import (
	"bytes"
	"context"

	sdkserviceclient "github.com/allinbits/sdk-service-meta/gen/grpc/sdk_utilities/client"
	sdkutilities "github.com/allinbits/sdk-service-meta/gen/sdk_utilities"

	"google.golang.org/grpc"

	tracelistener2 "github.com/allinbits/tracelistener"

	models "github.com/allinbits/demeris-backend-models/tracelistener"

	host "github.com/cosmos/cosmos-sdk/x/ibc/core/24-host"
	"go.uber.org/zap"
)

type clientCacheEntry struct {
	chainID  string
	clientID string
}

type ibcClientsProcessor struct {
	l            *zap.SugaredLogger
	grpcConn     *grpc.ClientConn
	clientsCache map[clientCacheEntry]models.IBCClientStateRow
}

func (*ibcClientsProcessor) TableSchema() string {
	return createClientsTable
}

func (b *ibcClientsProcessor) ModuleName() string {
	return "ibc_clients"
}

func (b *ibcClientsProcessor) FlushCache() []tracelistener2.WritebackOp {
	if len(b.clientsCache) == 0 {
		return nil
	}

	l := make([]models.DatabaseEntrier, 0, len(b.clientsCache))

	for _, c := range b.clientsCache {
		l = append(l, c)
	}

	b.clientsCache = map[clientCacheEntry]models.IBCClientStateRow{}

	return []tracelistener2.WritebackOp{
		{
			DatabaseExec: insertClient,
			Data:         l,
		},
	}
}

func (b *ibcClientsProcessor) OwnsKey(key []byte) bool {
	return bytes.Contains(key, []byte(host.KeyClientState))
}

func (b *ibcClientsProcessor) Process(data tracelistener2.TraceOperation) error {
	client := sdkserviceclient.NewClient(b.grpcConn)

	cc := sdkutilities.Client{
		IbcClientStateEndpoint: client.IbcClientState(),
	}

	payload := sdkutilities.TracePayload{
		Key:           data.Key,
		Value:         data.Value,
		OperationType: &data.Operation,
	}

	res, err := cc.IbcClientState(context.Background(), &sdkutilities.IbcClientStatePayload{
		Payload: []*sdkutilities.TracePayload{
			&payload,
		},
	})

	for _, r := range res {
		b.clientsCache[clientCacheEntry{
			chainID:  r.ChainID,
			clientID: r.ClientID,
		}] = models.IBCClientStateRow{
			ChainID:        r.ChainID,
			ClientID:       r.ClientID,
			LatestHeight:   r.LatestHeight,
			TrustingPeriod: r.TrustingPeriod,
		}
	}

	if err != nil {
		return unwindErrors(err)
	}

	return nil
}
