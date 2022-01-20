package processor

import (
	"bytes"

	models "github.com/allinbits/demeris-backend-models/tracelistener"

	"github.com/allinbits/tracelistener/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener/processor/datamarshaler"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
	"go.uber.org/zap"
)

type delegationCacheEntry struct {
	delegator string
	validator string
}

type delegationsProcessor struct {
	l                 *zap.SugaredLogger
	insertHeightCache map[delegationCacheEntry]models.DelegationRow
	deleteHeightCache map[delegationCacheEntry]models.DelegationRow
}

func (*delegationsProcessor) TableSchema() string {
	return createDelegationsTable
}

func (b *delegationsProcessor) ModuleName() string {
	return "delegations"
}

func (b *delegationsProcessor) FlushCache() []tracelistener.WritebackOp {
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

	return []tracelistener.WritebackOp{
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

func (b *delegationsProcessor) Process(data tracelistener.TraceOperation) error {
	res, err := datamarshaler.NewDataMarshaler(b.l).Delegations(data)
	if err != nil {
		return err
	}

	if data.Operation == tracelistener.DeleteOp.String() {
		b.deleteHeightCache[delegationCacheEntry{
			validator: res.Validator,
			delegator: res.Delegator,
		}] = models.DelegationRow{
			Delegator: res.Delegator,
			Validator: res.Validator,
		}

		return nil
	}

	b.insertHeightCache[delegationCacheEntry{
		validator: res.Validator,
		delegator: res.Delegator,
	}] = models.DelegationRow{
		Delegator:   res.Delegator,
		Validator:   res.Validator,
		Amount:      res.Amount,
		BlockHeight: data.BlockHeight,
	}

	return nil
}
