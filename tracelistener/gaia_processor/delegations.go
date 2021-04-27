package gaia_processor

import (
	"bytes"
	"encoding/hex"

	"github.com/allinbits/demeris-backend/tracelistener"
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
	l                 *zap.SugaredLogger
	insertHeightCache map[delegationCacheEntry]delegationWritebackPacket
	deleteHeightCache map[delegationCacheEntry]delegationWritebackPacket
}

func (*delegationsProcessor) TableSchema() string {
	return createDelegationsTable
}

func (b *delegationsProcessor) ModuleName() string {
	return "delegations"
}

func (b *delegationsProcessor) FlushCache() []tracelistener.WritebackOp {
	insert := make([]tracelistener.DatabaseEntrier, 0, len(b.insertHeightCache))
	delete := make([]tracelistener.DatabaseEntrier, 0, len(b.deleteHeightCache))

	if len(b.insertHeightCache) != 0 {
		for _, v := range b.insertHeightCache {
			insert = append(insert, v)
		}

		b.insertHeightCache = map[delegationCacheEntry]delegationWritebackPacket{}
	}

	if len(b.deleteHeightCache) == 0 && insert == nil {
		return nil
	}

	for _, v := range b.deleteHeightCache {
		delete = append(delete, v)
	}

	b.deleteHeightCache = map[delegationCacheEntry]delegationWritebackPacket{}

	return []tracelistener.WritebackOp{
		{
			DatabaseExec: insertDelegation,
			Data:         insert,
		},
		{
			DatabaseExec: deleteDelegation,
			Data:         delete,
		},
	}
}

func (b *delegationsProcessor) OwnsKey(key []byte) bool {
	return bytes.HasPrefix(key, types.DelegationKey)
}

func (b *delegationsProcessor) Process(data tracelistener.TraceOperation) error {
	if data.Operation == tracelistener.DeleteOp.String() {
		delegatorAddr := hex.EncodeToString(data.Key[1:21])
		validatorAddr := hex.EncodeToString(data.Key[21:41])
		b.l.Debugw("new delegation delete", "delegatorAddr", delegatorAddr, "validatorAddr", validatorAddr)

		b.deleteHeightCache[delegationCacheEntry{
			validator: validatorAddr,
			delegator: delegatorAddr,
		}] = delegationWritebackPacket{
			Delegator: delegatorAddr,
			Validator: validatorAddr,
		}

		return nil
	}

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

	b.insertHeightCache[delegationCacheEntry{
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
