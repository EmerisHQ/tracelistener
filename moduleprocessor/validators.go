package moduleprocessor

import (
	"bytes"
	"context"
	"time"

	"github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/allinbits/sdk-service-meta/tracelistener"

	sdkserviceclient "github.com/allinbits/sdk-service-meta/gen/grpc/sdk_utilities/client"
	sdkutilities "github.com/allinbits/sdk-service-meta/gen/sdk_utilities"
	"google.golang.org/grpc"

	tracelistener2 "github.com/allinbits/tracelistener"

	models "github.com/allinbits/demeris-backend-models/tracelistener"

	"go.uber.org/zap"
)

type validatorCacheEntry struct {
	operator string
}

type validatorsProcessor struct {
	l                 *zap.SugaredLogger
	grpcConn          *grpc.ClientConn
	insertHeightCache map[validatorCacheEntry]models.ValidatorRow
	deleteHeightCache map[validatorCacheEntry]models.ValidatorRow
}

func (*validatorsProcessor) TableSchema() string {
	return createValidatorsTable
}

func (b *validatorsProcessor) ModuleName() string {
	return "validators"
}

func (b *validatorsProcessor) FlushCache() []tracelistener2.WritebackOp {
	insert := make([]models.DatabaseEntrier, 0, len(b.insertHeightCache))
	deleteEntries := make([]models.DatabaseEntrier, 0, len(b.deleteHeightCache))

	if len(b.insertHeightCache) != 0 {
		for _, v := range b.insertHeightCache {
			insert = append(insert, v)
		}

		b.insertHeightCache = map[validatorCacheEntry]models.ValidatorRow{}
	}

	if len(b.deleteHeightCache) == 0 && insert == nil {
		return nil
	}

	for _, v := range b.deleteHeightCache {
		deleteEntries = append(deleteEntries, v)
	}

	b.deleteHeightCache = map[validatorCacheEntry]models.ValidatorRow{}

	return []tracelistener2.WritebackOp{
		{
			DatabaseExec: insertValidator,
			Data:         insert,
		},
		{
			DatabaseExec: deleteValidator,
			Data:         deleteEntries,
		},
	}
}

func (b *validatorsProcessor) OwnsKey(key []byte) bool {
	return bytes.HasPrefix(key, types.ValidatorsKey)
}

func (b *validatorsProcessor) Process(data tracelistener2.TraceOperation) error {
	client := sdkserviceclient.NewClient(b.grpcConn)

	cc := sdkutilities.Client{
		ValidatorEndpointEndpoint: client.ValidatorEndpoint(),
	}

	payload := sdkutilities.TracePayload{
		Key:           data.Key,
		Value:         data.Value,
		OperationType: &data.Operation,
	}

	res, err := cc.ValidatorEndpoint(context.Background(), &sdkutilities.ValidatorPayload{
		Payload: []*sdkutilities.TracePayload{
			&payload,
		},
	})

	for _, r := range res {
		switch r.Type {
		case tracelistener.TypeCreateValidator:
			n := models.ValidatorRow{
				OperatorAddress:      r.OperatorAddress,
				ConsensusPubKeyType:  r.ConsensusPubKeyType,
				ConsensusPubKeyValue: r.ConsensusPubKeyValue,
				Jailed:               r.Jailed,
				Status:               r.Status,
				Tokens:               r.Tokens,
				DelegatorShares:      r.DelegatorShares,
				Moniker:              r.Moniker,
				Identity:             r.Identity,
				Website:              r.Website,
				SecurityContact:      r.SecurityContact,
				Details:              r.Details,
				UnbondingHeight:      r.UnbondingHeight,
				UnbondingTime:        time.Unix(r.UnbondingTime, 0).String(),
				CommissionRate:       r.CommissionRate,
				MaxRate:              r.MaxRate,
				MaxChangeRate:        r.MaxChangeRate,
				UpdateTime:           r.UpdateTime,
				MinSelfDelegation:    r.MinSelfDelegation,
			}

			b.insertHeightCache[validatorCacheEntry{
				operator: r.OperatorAddress,
			}] = n
		case tracelistener.TypeDeleteValidator:
			b.deleteHeightCache[validatorCacheEntry{
				operator: r.OperatorAddress,
			}] = models.ValidatorRow{
				OperatorAddress: r.OperatorAddress,
			}
		}
	}

	if err != nil {
		return unwindErrors(err)
	}

	return nil
}
