package processor

import (
	"bytes"
	"sync"

	"github.com/emerishq/tracelistener/tracelistener/processor/datamarshaler"
	"github.com/emerishq/tracelistener/tracelistener/tables"
	"go.uber.org/zap"

	models "github.com/emerishq/demeris-backend-models/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener"
)

var validatorsTable = tables.NewValidatorsTable("tracelistener.validators")

type validatorCacheEntry struct {
	operator string
}
type validatorsProcessor struct {
	l                     *zap.SugaredLogger
	insertValidatorsCache map[validatorCacheEntry]models.ValidatorRow
	deleteValidatorsCache map[validatorCacheEntry]models.ValidatorRow
	m                     sync.Mutex
}

var (
	addValAddressColumn = `ALTER TABLE ` + validatorsTable.Name() + ` ADD COLUMN IF NOT EXISTS validator_address text DEFAULT '';`
)

func (*validatorsProcessor) Migrations() []string {
	return append(validatorsTable.Migrations(), addValAddressColumn)
}

func (b *validatorsProcessor) ModuleName() string {
	return "validators"
}

func (b *validatorsProcessor) SDKModuleName() tracelistener.SDKModuleName {
	return tracelistener.Staking
}

func (b *validatorsProcessor) InsertStatement() string {
	return validatorsTable.Insert()
}

func (b *validatorsProcessor) UpsertStatement() string {
	return validatorsTable.Upsert()
}

func (b *validatorsProcessor) DeleteStatement() string {
	return validatorsTable.Delete()
}

func (b *validatorsProcessor) FlushCache() []tracelistener.WritebackOp {
	b.m.Lock()
	defer b.m.Unlock()

	insert := make([]models.DatabaseEntrier, 0, len(b.insertValidatorsCache))

	// pre-allocate wbOp as follows:
	// - 1 capacity unit for an eventual insert op
	// - n capacity units for each element in deleteHeightCache
	writebackOp := make([]tracelistener.WritebackOp, 0, 1+len(b.deleteValidatorsCache))

	if len(b.insertValidatorsCache) != 0 {
		for _, v := range b.insertValidatorsCache {
			insert = append(insert, v)
		}

		b.insertValidatorsCache = map[validatorCacheEntry]models.ValidatorRow{}
	}

	writebackOp = append(writebackOp, tracelistener.WritebackOp{
		Type: tracelistener.Write,
		Data: insert,
	})

	if len(b.deleteValidatorsCache) == 0 && len(insert) == 0 {
		return nil
	}

	for _, v := range b.deleteValidatorsCache {
		writebackOp = append(writebackOp, tracelistener.WritebackOp{
			Type: tracelistener.Delete,
			Data: []models.DatabaseEntrier{v},
		})
	}

	b.deleteValidatorsCache = map[validatorCacheEntry]models.ValidatorRow{}

	return writebackOp
}
func (b *validatorsProcessor) OwnsKey(key []byte) bool {
	ret := bytes.HasPrefix(key, datamarshaler.ValidatorsKey)
	return ret
}

func (b *validatorsProcessor) Process(data tracelistener.TraceOperation) error {
	b.m.Lock()
	defer b.m.Unlock()

	res, err := datamarshaler.NewDataMarshaler(b.l).Validators(data)
	if err != nil {
		return err
	}

	if data.Operation == tracelistener.DeleteOp.String() {
		b.deleteValidatorsCache[validatorCacheEntry{
			operator: res.OperatorAddress,
		}] = res

		return nil
	}

	b.insertValidatorsCache[validatorCacheEntry{
		operator: res.OperatorAddress,
	}] = res

	return nil
}
