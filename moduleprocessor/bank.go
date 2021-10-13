package moduleprocessor

import (
	"bytes"
	"context"

	"google.golang.org/grpc"

	sdkutilities "github.com/allinbits/sdk-service-meta/gen/sdk_utilities"

	"github.com/allinbits/tracelistener"

	models "github.com/allinbits/demeris-backend-models/tracelistener"

	"go.uber.org/zap"

	"github.com/cosmos/cosmos-sdk/x/bank/types"

	sdkserviceclient "github.com/allinbits/sdk-service-meta/gen/grpc/sdk_utilities/client"
)

type bankCacheEntry struct {
	address string
	denom   string
}

type bankProcessor struct {
	l           *zap.SugaredLogger
	grpcConn    *grpc.ClientConn
	heightCache map[bankCacheEntry]models.BalanceRow
}

func (*bankProcessor) TableSchema() string {
	return createBalancesTable
}

func (b *bankProcessor) ModuleName() string {
	return "bank"
}

func (b *bankProcessor) FlushCache() []tracelistener.WritebackOp {
	if len(b.heightCache) == 0 {
		return nil
	}

	l := make([]models.DatabaseEntrier, 0, len(b.heightCache))

	for _, v := range b.heightCache {
		l = append(l, v)
	}

	b.heightCache = map[bankCacheEntry]models.BalanceRow{}

	return []tracelistener.WritebackOp{
		{
			DatabaseExec: insertBalance,
			Data:         l,
		},
	}
}

func (b *bankProcessor) OwnsKey(key []byte) bool {
	return bytes.HasPrefix(key, types.BalancesPrefix)
}

func (b *bankProcessor) Process(data tracelistener.TraceOperation) error {
	client := sdkserviceclient.NewClient(b.grpcConn)

	cc := sdkutilities.Client{
		BankEndpoint: client.Bank(),
	}

	payload := sdkutilities.TracePayload{
		Key:           data.Key,
		Value:         data.Value,
		OperationType: &data.Operation,
	}

	res, err := cc.Bank(context.Background(), &sdkutilities.BankPayload{
		Payload: []*sdkutilities.TracePayload{
			&payload,
		},
	})

	for _, r := range res {
		b.heightCache[bankCacheEntry{
			address: r.Address,
			denom:   r.Denom,
		}] = models.BalanceRow{
			Address:     r.Address,
			Amount:      r.Amount,
			Denom:       r.Denom,
			BlockHeight: data.BlockHeight,
		}
	}

	if err != nil {
		return unwindErrors(err)
	}

	return nil
}
