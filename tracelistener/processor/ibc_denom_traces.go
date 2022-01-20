package processor

import (
	"bytes"
	"encoding/hex"
	"fmt"

	models "github.com/allinbits/demeris-backend-models/tracelistener"

	"github.com/allinbits/tracelistener/tracelistener"
	transferTypes "github.com/cosmos/cosmos-sdk/x/ibc/applications/transfer/types"
	"go.uber.org/zap"
)

type ibcDenomTracesProcessor struct {
	l                *zap.SugaredLogger
	denomTracesCache map[string]models.IBCDenomTraceRow
}

func (*ibcDenomTracesProcessor) TableSchema() string {
	return createDenomTracesTable
}

func (b *ibcDenomTracesProcessor) ModuleName() string {
	return "ibc_denom_traces"
}

func (b *ibcDenomTracesProcessor) FlushCache() []tracelistener.WritebackOp {
	if len(b.denomTracesCache) == 0 {
		return nil
	}

	l := make([]models.DatabaseEntrier, 0, len(b.denomTracesCache))

	for _, c := range b.denomTracesCache {
		l = append(l, c)
	}

	b.denomTracesCache = map[string]models.IBCDenomTraceRow{}

	return []tracelistener.WritebackOp{
		{
			DatabaseExec: insertDenomTrace,
			Data:         l,
		},
	}
}

func (b *ibcDenomTracesProcessor) OwnsKey(key []byte) bool {
	return bytes.HasPrefix(key, transferTypes.DenomTraceKey)
}

func (b *ibcDenomTracesProcessor) Process(data tracelistener.TraceOperation) error {
	b.l.Debugw("beginning denom trace processor", "key", string(data.Key), "value", string(data.Value))

	dt := transferTypes.DenomTrace{}
	if err := p.cdc.UnmarshalBinaryBare(data.Value, &dt); err != nil {
		return err
	}

	if err := dt.Validate(); err != nil {
		b.l.Debugw("found a denom trace that isn't ICS20 compliant", "denom trace", dt, "error", err)
		return fmt.Errorf("denom trace validation failed, %w", err)
	}

	if dt.BaseDenom == "" {
		b.l.Debugw("ignoring since it's not a denom trace")
		return nil
	}

	hash := hex.EncodeToString(dt.Hash())

	newObj := models.IBCDenomTraceRow{
		Path:      dt.Path,
		BaseDenom: dt.BaseDenom,
		Hash:      hash,
	}

	b.l.Debugw("denom trace unmarshaled", "object", newObj)

	b.denomTracesCache[hash] = newObj
	return nil
}
