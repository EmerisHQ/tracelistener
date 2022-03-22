package processor

import (
	"bytes"
	"sync"

	"github.com/emerishq/tracelistener/tracelistener/processor/datamarshaler"
	"go.uber.org/zap"

	models "github.com/emerishq/demeris-backend-models/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener"
)

type unbondingDelegationCacheEntry struct {
	delegator string
	validator string
}

type unbondingDelegationsProcessor struct {
	l                 *zap.SugaredLogger
	insertHeightCache map[unbondingDelegationCacheEntry]models.UnbondingDelegationRow
	deleteHeightCache map[unbondingDelegationCacheEntry]models.UnbondingDelegationRow
	m                 sync.Mutex
}

func (*unbondingDelegationsProcessor) TableSchema() string {
	return createUnbondingDelegationsTable
}

func (b *unbondingDelegationsProcessor) ModuleName() string {
	return "unbonding_delegations"
}

func (b *unbondingDelegationsProcessor) SDKModuleName() tracelistener.SDKModuleName {
	return tracelistener.Staking
}

func (b *unbondingDelegationsProcessor) FlushCache(upsert bool) []tracelistener.WritebackOp {
	b.m.Lock()
	defer b.m.Unlock()

	insert := make([]models.DatabaseEntrier, 0, len(b.insertHeightCache))

	// pre-allocate wbOp as follows:
	// - 1 capacity unit for an eventual insert op
	// - n capacity units for each element in deleteHeightCache
	writebackOp := make([]tracelistener.WritebackOp, 0, 1+len(b.deleteHeightCache))

	if len(b.insertHeightCache) != 0 {
		for _, v := range b.insertHeightCache {
			insert = append(insert, v)
		}

		b.insertHeightCache = map[unbondingDelegationCacheEntry]models.UnbondingDelegationRow{}
	}

	writebackOp = append(writebackOp, tracelistener.WritebackOp{
		Type: insertUnbondingDelegation,
		Data: insert,
	})

	if len(b.deleteHeightCache) == 0 && len(insert) == 0 {
		return nil
	}

	for _, v := range b.deleteHeightCache {
		writebackOp = append(writebackOp, tracelistener.WritebackOp{
			Type: deleteUnbondingDelegation,
			Data: []models.DatabaseEntrier{v},
		})
	}

	b.deleteHeightCache = map[unbondingDelegationCacheEntry]models.UnbondingDelegationRow{}

	return writebackOp
}

func (b *unbondingDelegationsProcessor) OwnsKey(key []byte) bool {
	for _, rkey := range datamarshaler.UnbondingDelegationKeys {
		if bytes.HasPrefix(key, rkey) {
			return true
		}
	}

	return false
}

func (b *unbondingDelegationsProcessor) Process(data tracelistener.TraceOperation) error {
	b.m.Lock()
	defer b.m.Unlock()

	res, err := datamarshaler.NewDataMarshaler(b.l).UnbondingDelegations(data)
	if err != nil {
		return err
	}

	if res.Delegator == "" && res.Validator == "" {
		return nil // case in which this is an error operation, but the key wasn't UnbondingDelegationByValidatorKey
	}

	if data.Operation == tracelistener.DeleteOp.String() {
		b.deleteHeightCache[unbondingDelegationCacheEntry{
			validator: res.Validator,
			delegator: res.Delegator,
		}] = models.UnbondingDelegationRow{
			Delegator: res.Delegator,
			Validator: res.Validator,
		}

		return nil
	}

	b.insertHeightCache[unbondingDelegationCacheEntry{
		validator: res.Validator,
		delegator: res.Delegator,
	}] = models.UnbondingDelegationRow{
		Delegator: res.Delegator,
		Validator: res.Validator,
		Entries:   res.Entries,
	}

	return nil
}
