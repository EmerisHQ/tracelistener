package gaia_processor

import (
	"bytes"
	"encoding/hex"

	"github.com/allinbits/tracelistener"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
	"go.uber.org/zap"
)

type delegationWritebackPacket struct {
	tracelistener.BasicDatabaseEntry

	Delegator   string `db:"delegator_address" json:"delegator"`
	Validator   string `db:"validator_address" json:"validator"`
	Amount      string `db:"amount" json:"amount"`
	BlockHeight uint64 `db:"height" json:"block_height"`
}

func (b delegationWritebackPacket) WithChainName(cn string) tracelistener.DatabaseEntrier {
	b.ChainName = cn
	return b
}

type delegationCacheEntry struct {
	delegator string
	validator string
}

type delegationsProcessor struct {
	l           *zap.SugaredLogger
	heightCache map[delegationCacheEntry]delegationWritebackPacket
}

func (*delegationsProcessor) TableSchema() string {
	return createDelegationsTable
}

func (b *delegationsProcessor) ModuleName() string {
	return "delegations"
}

func (b *delegationsProcessor) FlushCache() tracelistener.WritebackOp {
	if len(b.heightCache) == 0 {
		return tracelistener.WritebackOp{}
	}

	l := make([]tracelistener.DatabaseEntrier, 0, len(b.heightCache))

	for _, v := range b.heightCache {
		l = append(l, v)
	}

	b.heightCache = map[delegationCacheEntry]delegationWritebackPacket{}

	return tracelistener.WritebackOp{
		DatabaseExec: insertDelegation,
		Data:         l,
	}
}

func (b *delegationsProcessor) OwnsKey(key []byte) bool {
	return bytes.HasPrefix(key, types.DelegationKey)
}

func (b *delegationsProcessor) Process(data tracelistener.TraceOperation) error {
	delegation := types.Delegation{}

	if err := p.cdc.UnmarshalBinaryBare(data.Value, &delegation); err != nil {
		return err
	}

	delegator, err := b32Hex(delegation.DelegatorAddress)
	if err != nil {
		return err
	}

	validator, err := b32Hex(delegation.ValidatorAddress)
	if err != nil {
		return err
	}

	b.l.Debugw("new delegation write",
		"operation", data.Operation,
		"delegator", delegator,
		"validator", "validator",
		"amount", delegation.Shares.String(),
		"height", data.BlockHeight,
		"txHash", data.TxHash,
	)

	b.heightCache[delegationCacheEntry{
		validator: validator,
		delegator: delegator,
	}] = delegationWritebackPacket{
		Delegator:   delegator,
		Validator:   validator,
		Amount:      delegation.Shares.String(),
		BlockHeight: data.BlockHeight,
	}

	return nil
}

func b32Hex(s string) (string, error) {
	_, b, err := bech32.DecodeAndConvert(s)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}
