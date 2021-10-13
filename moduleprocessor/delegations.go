package moduleprocessor

import (
	"bytes"
	"context"

	"github.com/allinbits/sdk-service-meta/tracelistener"

	sdkserviceclient "github.com/allinbits/sdk-service-meta/gen/grpc/sdk_utilities/client"
	sdkutilities "github.com/allinbits/sdk-service-meta/gen/sdk_utilities"
	"google.golang.org/grpc"

	tracelistener2 "github.com/allinbits/tracelistener"

	models "github.com/allinbits/demeris-backend-models/tracelistener"

	"github.com/cosmos/cosmos-sdk/x/staking/types"
	"go.uber.org/zap"
)

type delegationCacheEntry struct {
	delegator string
	validator string
}

type delegationsProcessor struct {
	l                 *zap.SugaredLogger
	grpcConn          *grpc.ClientConn
	insertHeightCache map[delegationCacheEntry]models.DelegationRow
	deleteHeightCache map[delegationCacheEntry]models.DelegationRow
}

func (*delegationsProcessor) TableSchema() string {
	return createDelegationsTable
}

func (b *delegationsProcessor) ModuleName() string {
	return "delegations"
}

func (b *delegationsProcessor) FlushCache() []tracelistener2.WritebackOp {
	insert := make([]models.DatabaseEntrier, 0, len(b.insertHeightCache))
	deleteEntries := make([]models.DatabaseEntrier, 0, len(b.deleteHeightCache))

	if len(b.insertHeightCache) != 0 {
		for _, v := range b.insertHeightCache {
			insert = append(insert, v)
		}

		b.insertHeightCache = map[delegationCacheEntry]models.DelegationRow{}
	}

	if len(b.deleteHeightCache) == 0 && insert == nil {
		return nil
	}

	for _, v := range b.deleteHeightCache {
		deleteEntries = append(deleteEntries, v)
	}

	b.deleteHeightCache = map[delegationCacheEntry]models.DelegationRow{}

	return []tracelistener2.WritebackOp{
		{
			DatabaseExec: insertDelegation,
			Data:         insert,
		},
		{
			DatabaseExec: deleteDelegation,
			Data:         deleteEntries,
		},
	}
}

func (b *delegationsProcessor) OwnsKey(key []byte) bool {
	return bytes.HasPrefix(key, types.DelegationKey)
}

func (b *delegationsProcessor) Process(data tracelistener2.TraceOperation) error {
	client := sdkserviceclient.NewClient(b.grpcConn)

	cc := sdkutilities.Client{
		DelegationEndpointEndpoint: client.DelegationEndpoint(),
	}

	payload := sdkutilities.TracePayload{
		Key:           data.Key,
		Value:         data.Value,
		OperationType: &data.Operation,
	}

	res, err := cc.DelegationEndpoint(context.Background(), &sdkutilities.DelegationPayload{
		Payload: []*sdkutilities.TracePayload{
			&payload,
		},
	})

	for _, r := range res {
		switch r.Type {
		case tracelistener.TypeCreateDelegation:
			b.insertHeightCache[delegationCacheEntry{
				delegator: r.Delegator,
				validator: r.Validator,
			}] = models.DelegationRow{
				Delegator: r.Delegator,
				Validator: r.Validator,
				Amount:    r.Amount,
			}
		case tracelistener.TypeDeleteDelegation:
			b.deleteHeightCache[delegationCacheEntry{
				delegator: r.Delegator,
				validator: r.Validator,
			}] = models.DelegationRow{
				Delegator: r.Delegator,
				Validator: r.Validator,
				Amount:    r.Amount,
			}
		}
	}

	if err != nil {
		return unwindErrors(err)
	}

	return nil
}
