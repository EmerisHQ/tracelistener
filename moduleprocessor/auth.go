package moduleprocessor

import (
	"bytes"
	"context"

	sdkserviceclient "github.com/allinbits/sdk-service-meta/gen/grpc/sdk_utilities/client"
	sdkutilities "github.com/allinbits/sdk-service-meta/gen/sdk_utilities"

	tracelistener2 "github.com/allinbits/tracelistener"
	"google.golang.org/grpc"

	"github.com/allinbits/tracelistener/models"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	"go.uber.org/zap"
)

type authCacheEntry struct {
	address   string
	accNumber uint64
}

type authProcessor struct {
	l           *zap.SugaredLogger
	grpcConn    *grpc.ClientConn
	heightCache map[authCacheEntry]models.AuthRow
}

func (*authProcessor) TableSchema() string {
	return createAuthTable
}

func (b *authProcessor) ModuleName() string {
	return "auth"
}

func (b *authProcessor) FlushCache() []tracelistener2.WritebackOp {
	if len(b.heightCache) == 0 {
		return nil
	}

	l := make([]models.DatabaseEntrier, 0, len(b.heightCache))

	for _, v := range b.heightCache {
		l = append(l, v)
	}

	b.heightCache = map[authCacheEntry]models.AuthRow{}

	return []tracelistener2.WritebackOp{
		{
			DatabaseExec: insertAuth,
			Data:         l,
		},
	}
}

func (b *authProcessor) OwnsKey(key []byte) bool {
	return bytes.HasPrefix(key, types.AddressStoreKeyPrefix)
}

func (b *authProcessor) Process(data tracelistener2.TraceOperation) error {
	client := sdkserviceclient.NewClient(b.grpcConn)

	cc := sdkutilities.Client{
		AuthEndpointEndpoint: client.AuthEndpoint(),
	}

	payload := sdkutilities.TracePayload{
		Key:           data.Key,
		Value:         data.Value,
		OperationType: &data.Operation,
	}

	res, err := cc.AuthEndpoint(context.Background(), &sdkutilities.AuthPayload{
		Payload: []*sdkutilities.TracePayload{
			&payload,
		},
	})

	for _, r := range res {
		b.heightCache[authCacheEntry{
			address:   r.Address,
			accNumber: r.AccountNumber,
		}] = models.AuthRow{
			Address:        r.Address,
			SequenceNumber: r.SequenceNumber,
			AccountNumber:  r.AccountNumber,
		}
	}

	if err != nil {
		return unwindErrors(err)
	}

	return nil
}
