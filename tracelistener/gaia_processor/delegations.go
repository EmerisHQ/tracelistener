package gaia_processor

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/allinbits/demeris-backend/models"

	"github.com/allinbits/demeris-backend/tracelistener"
	"github.com/cosmos/cosmos-sdk/types/bech32"
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
	if data.Operation == tracelistener.DeleteOp.String() {
		if len(data.Key) < 41 { // 20 bytes by address, 1 prefix = 2*20 + 1
			return nil // found probably liquidity stuff being deleted
		}

		delegatorAddr := hex.EncodeToString(data.Key[1:21])
		validatorAddr := hex.EncodeToString(data.Key[21:41])
		b.l.Debugw("new delegation delete", "delegatorAddr", delegatorAddr, "validatorAddr", validatorAddr)

		b.deleteHeightCache[delegationCacheEntry{
			validator: validatorAddr,
			delegator: delegatorAddr,
		}] = models.DelegationRow{
			Delegator: delegatorAddr,
			Validator: validatorAddr,
		}

		return nil
	}

	delegation := types.Delegation{}

	if err := p.cdc.Unmarshal(data.Value, &delegation); err != nil {
		return err
	}

	delegator, err := b32Hex(delegation.DelegatorAddress)
	if err != nil {
		return fmt.Errorf("cannot convert delegator address from bech32 to hex, %w", err)
	}

	validator, err := b32Hex(delegation.ValidatorAddress)
	if err != nil {
		return fmt.Errorf("cannot convert validator address from bech32 to hex, %w", err)
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
	}] = models.DelegationRow{
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
