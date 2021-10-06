package moduleprocessor

import (
	"bytes"
	"context"

	sdkserviceclient "github.com/allinbits/sdk-service-meta/gen/grpc/sdk_utilities/client"
	sdkutilities "github.com/allinbits/sdk-service-meta/gen/sdk_utilities"

	tracelistener2 "github.com/allinbits/tracelistener"
	"github.com/allinbits/tracelistener/models"
	"google.golang.org/grpc"

	host "github.com/cosmos/cosmos-sdk/x/ibc/core/24-host"
	"go.uber.org/zap"
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
	grpcConn         *grpc.ClientConn
	connectionsCache map[connectionCacheEntry]models.IBCConnectionRow
}

func (*ibcConnectionsProcessor) TableSchema() string {
	return createConnectionsTable
}

func (b *ibcConnectionsProcessor) ModuleName() string {
	return "ibc_connections"
}

func (b *ibcConnectionsProcessor) FlushCache() []tracelistener2.WritebackOp {
	if len(b.connectionsCache) == 0 {
		return nil
	}

	l := make([]models.DatabaseEntrier, 0, len(b.connectionsCache))

	for _, c := range b.connectionsCache {
		l = append(l, c)
	}

	b.connectionsCache = map[connectionCacheEntry]models.IBCConnectionRow{}

	return []tracelistener2.WritebackOp{
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

func (b *ibcConnectionsProcessor) Process(data tracelistener2.TraceOperation) error {
	client := sdkserviceclient.NewClient(b.grpcConn)

	cc := sdkutilities.Client{
		IbcConnectionEndpoint: client.IbcConnection(),
	}

	payload := sdkutilities.TracePayload{
		Key:           data.Key,
		Value:         data.Value,
		OperationType: &data.Operation,
	}

	res, err := cc.IbcConnection(context.Background(), &sdkutilities.IbcConnectionPayload{
		Payload: []*sdkutilities.TracePayload{
			&payload,
		},
	})

	for _, r := range res {
		b.connectionsCache[connectionCacheEntry{
			connectionID: r.ConnectionID,
			clientID:     r.ClientID,
		}] = models.IBCConnectionRow{
			ConnectionID:        r.ConnectionID,
			ClientID:            r.ClientID,
			State:               r.State,
			CounterConnectionID: r.CounterConnectionID,
			CounterClientID:     r.CounterClientID,
		}
	}

	if err != nil {
		return unwindErrors(err)
	}

	return nil
}
