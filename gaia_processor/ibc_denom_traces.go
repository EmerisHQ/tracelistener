package gaia_processor

import (
	"bytes"
	"encoding/hex"

	"github.com/allinbits/tracelistener"
	transferTypes "github.com/cosmos/cosmos-sdk/x/ibc/applications/transfer/types"
	"go.uber.org/zap"
)

type denomTracesWritebackPacket struct {
	tracelistener.BasicDatabaseEntry

	Path      string `json:"path" db:"path"`
	BaseDenom string `json:"base_denom" db:"base_denom"`
	Hash      string `json:"hash" db:"hash"`
}

func (c denomTracesWritebackPacket) WithChainName(cn string) tracelistener.DatabaseEntrier {
	c.ChainName = cn
	return c
}

type ibcDenomTracesProcessor struct {
	l                *zap.SugaredLogger
	denomTracesCache map[string]denomTracesWritebackPacket
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

	l := make([]tracelistener.DatabaseEntrier, 0, len(b.denomTracesCache))

	for _, c := range b.denomTracesCache {
		l = append(l, c)
	}

	b.denomTracesCache = map[string]denomTracesWritebackPacket{}

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
	dt := transferTypes.DenomTrace{}
	if err := p.cdc.UnmarshalBinaryBare(data.Value, &dt); err != nil {
		return err
	}

	b.denomTracesCache[dt.Path] = denomTracesWritebackPacket{
		Path:      dt.Path,
		BaseDenom: dt.BaseDenom,
		Hash:      hex.EncodeToString(dt.Hash()),
	}
	return nil
}
