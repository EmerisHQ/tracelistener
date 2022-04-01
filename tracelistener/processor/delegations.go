package processor

import (
	"bytes"
	"sync"

	"github.com/emerishq/tracelistener/tracelistener/processor/datamarshaler"
	"go.uber.org/zap"

	models "github.com/emerishq/demeris-backend-models/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener"
)

type delegationCacheEntry struct {
	delegator string
	validator string
}

type delegationsProcessor struct {
	l                 *zap.SugaredLogger
	insertHeightCache map[delegationCacheEntry]models.DelegationRow
	deleteHeightCache map[delegationCacheEntry]models.DelegationRow
	m                 sync.Mutex
}

func (*delegationsProcessor) Migrations() []string {
	return []string{createDelegationsTable}
}

func (b *delegationsProcessor) ModuleName() string {
	return "delegations"
}

func (b *delegationsProcessor) SDKModuleName() tracelistener.SDKModuleName {
	return tracelistener.Staking
}

func (b *delegationsProcessor) UpsertStatement() string {
	return upsertDelegation
}

func (b *delegationsProcessor) InsertStatement() string {
	return insertDelegation
}

func (b *delegationsProcessor) DeleteStatement() string {
	return deleteDelegation
}

func (b *delegationsProcessor) FlushCache() []tracelistener.WritebackOp {
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

		b.insertHeightCache = map[delegationCacheEntry]models.DelegationRow{}
	}

	writebackOp = append(writebackOp, tracelistener.WritebackOp{
		Type: tracelistener.Write,
		Data: insert,
	})

	if len(b.deleteHeightCache) == 0 && len(insert) == 0 {
		return nil
	}

	for _, v := range b.deleteHeightCache {
		writebackOp = append(writebackOp, tracelistener.WritebackOp{
			Type: tracelistener.Delete,
			Data: []models.DatabaseEntrier{v},
		})
	}

	b.deleteHeightCache = map[delegationCacheEntry]models.DelegationRow{}

	return writebackOp
}

func (b *delegationsProcessor) OwnsKey(key []byte) bool {
	return bytes.HasPrefix(key, datamarshaler.DelegationKey)
}

func (b *delegationsProcessor) Process(data tracelistener.TraceOperation) error {
	b.m.Lock()
	defer b.m.Unlock()

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
