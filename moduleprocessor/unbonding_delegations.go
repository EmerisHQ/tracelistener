package moduleprocessor

import (
	"bytes"
	"context"
	"time"

	"github.com/allinbits/sdk-service-meta/tracelistener"

	sdkserviceclient "github.com/allinbits/sdk-service-meta/gen/grpc/sdk_utilities/client"
	sdkutilities "github.com/allinbits/sdk-service-meta/gen/sdk_utilities"
	"google.golang.org/grpc"

	tracelistener2 "github.com/allinbits/tracelistener"

	"github.com/allinbits/tracelistener/models"

	"github.com/cosmos/cosmos-sdk/x/staking/types"
	"go.uber.org/zap"
)

type unbondingDelegationCacheEntry struct {
	delegator string
	validator string
}

type unbondingDelegationsProcessor struct {
	l                 *zap.SugaredLogger
	grpcConn          *grpc.ClientConn
	insertHeightCache map[unbondingDelegationCacheEntry]models.UnbondingDelegationRow
	deleteHeightCache map[unbondingDelegationCacheEntry]models.UnbondingDelegationRow
}

func (*unbondingDelegationsProcessor) TableSchema() string {
	return createUnbondingDelegationsTable
}

func (b *unbondingDelegationsProcessor) ModuleName() string {
	return "unbonding_delegations"
}

func (b *unbondingDelegationsProcessor) FlushCache() []tracelistener2.WritebackOp {
	insert := make([]models.DatabaseEntrier, 0, len(b.insertHeightCache))
	deleteEntries := make([]models.DatabaseEntrier, 0, len(b.deleteHeightCache))

	if len(b.insertHeightCache) != 0 {
		for _, v := range b.insertHeightCache {
			insert = append(insert, v)
		}

		b.insertHeightCache = map[unbondingDelegationCacheEntry]models.UnbondingDelegationRow{}
	}

	if len(b.deleteHeightCache) == 0 && insert == nil {
		return nil
	}

	for _, v := range b.deleteHeightCache {
		deleteEntries = append(deleteEntries, v)
	}

	b.deleteHeightCache = map[unbondingDelegationCacheEntry]models.UnbondingDelegationRow{}

	return []tracelistener2.WritebackOp{
		{
			DatabaseExec: insertUnbondingDelegation,
			Data:         insert,
		},
		{
			DatabaseExec: deleteUnbondingDelegation,
			Data:         deleteEntries,
		},
	}
}

func (b *unbondingDelegationsProcessor) OwnsKey(key []byte) bool {
	return bytes.HasPrefix(key, types.UnbondingDelegationKey)
}

func (b *unbondingDelegationsProcessor) Process(data tracelistener2.TraceOperation) error {
	client := sdkserviceclient.NewClient(b.grpcConn)

	cc := sdkutilities.Client{
		UnbondingDelegationEndpointEndpoint: client.UnbondingDelegationEndpoint(),
	}

	payload := sdkutilities.TracePayload{
		Key:           data.Key,
		Value:         data.Value,
		OperationType: &data.Operation,
	}

	res, err := cc.UnbondingDelegationEndpoint(context.Background(), &sdkutilities.UnbondingDelegationPayload{
		Payload: []*sdkutilities.TracePayload{
			&payload,
		},
	})

	for _, r := range res {
		switch r.Type {
		case tracelistener.TypeCreateUnbondingDelegation:
			n := models.UnbondingDelegationRow{
				Delegator: r.Delegator,
				Validator: r.Validator,
			}

			for _, ee := range r.Entries {
				n.Entries = append(n.Entries, models.UnbondingDelegationEntry{
					Balance:        ee.Balance,
					InitialBalance: ee.InitialBalance,
					CreationHeight: ee.CreationHeight,
					CompletionTime: time.Unix(ee.CompletionTime, 0).String(),
				})
			}

			b.insertHeightCache[unbondingDelegationCacheEntry{
				delegator: r.Delegator,
				validator: r.Validator,
			}] = n
		case tracelistener.TypeDeleteUnbondingDelegation:
			b.deleteHeightCache[unbondingDelegationCacheEntry{
				delegator: r.Delegator,
				validator: r.Validator,
			}] = models.UnbondingDelegationRow{
				Delegator: r.Delegator,
				Validator: r.Validator,
			}
		}
	}

	if err != nil {
		return unwindErrors(err)
	}

	return nil
}
