package processor

import (
	"bytes"
	"sync"

	"github.com/allinbits/tracelistener/tracelistener/processor/datamarshaler"
	"go.uber.org/zap"

	models "github.com/allinbits/demeris-backend-models/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener"
)

type validatorCacheEntry struct {
	operator string
}
type validatorsProcessor struct {
	l                     *zap.SugaredLogger
	insertValidatorsCache map[validatorCacheEntry]models.ValidatorRow
	deleteValidatorsCache map[validatorCacheEntry]models.ValidatorRow
	m                     sync.Mutex
}

func (*validatorsProcessor) TableSchema() string {
	return createValidatorsTable
}

func (b *validatorsProcessor) ModuleName() string {
	return "validators"
}

func (b *validatorsProcessor) FlushCache() []tracelistener.WritebackOp {
	b.m.Lock()
	defer b.m.Unlock()

	if len(b.insertValidatorsCache) == 0 && len(b.deleteValidatorsCache) == 0 {
		return nil
	}

	insertValidators := make([]models.DatabaseEntrier, 0, len(b.insertValidatorsCache))
	deleteValidators := make([]models.DatabaseEntrier, 0, len(b.deleteValidatorsCache))

	if len(b.insertValidatorsCache) != 0 {
		for _, v := range b.insertValidatorsCache {
			insertValidators = append(insertValidators, v)
		}
	}

	b.insertValidatorsCache = map[validatorCacheEntry]models.ValidatorRow{}

	if len(b.deleteValidatorsCache) != 0 {
		for _, v := range b.deleteValidatorsCache {
			deleteValidators = append(deleteValidators, v)
		}
	}

	b.deleteValidatorsCache = map[validatorCacheEntry]models.ValidatorRow{}

	return []tracelistener.WritebackOp{
		{
			DatabaseExec: insertValidator,
			Data:         insertValidators,
		},
		{
			DatabaseExec: deleteValidator,
			Data:         deleteValidators,
		},
	}
}
func (b *validatorsProcessor) OwnsKey(key []byte) bool {
	return bytes.HasPrefix(key, datamarshaler.ValidatorsKey)
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
