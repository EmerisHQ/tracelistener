package gaia_processor

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/allinbits/tracelistener/models"

	"github.com/allinbits/tracelistener/tracelistener"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
	"go.uber.org/zap"
)

type unbondingDelegationCacheEntry struct {
	delegator string
	validator string
}

type unbondingDelegationsProcessor struct {
	l                 *zap.SugaredLogger
	insertHeightCache map[unbondingDelegationCacheEntry]models.UnbondingDelegationRow
	deleteHeightCache map[unbondingDelegationCacheEntry]models.UnbondingDelegationRow
}

func (*unbondingDelegationsProcessor) TableSchema() string {
	return createUnbondingDelegationsTable
}

func (b *unbondingDelegationsProcessor) ModuleName() string {
	return "unbonding_delegations"
}

func (b *unbondingDelegationsProcessor) FlushCache() []tracelistener.WritebackOp {
	insert := make([]models.DatabaseEntrier, 0, len(b.insertHeightCache))
	deleteEntries := make([]models.DatabaseEntrier, 0, len(b.deleteHeightCache))

	if len(b.insertHeightCache) != 0 {
		for _, v := range b.insertHeightCache {
			insert = append(insert, v)
		}

		b.insertHeightCache = map[unbondingDelegationCacheEntry]models.UnbondingDelegationRow{}
	}

	if len(b.deleteHeightCache) == 0 && insert == nil {
		return nil
	}

	for _, v := range b.deleteHeightCache {
		deleteEntries = append(deleteEntries, v)
	}

	b.deleteHeightCache = map[unbondingDelegationCacheEntry]models.UnbondingDelegationRow{}

	return []tracelistener.WritebackOp{
		{
			DatabaseExec: insertUnbondingDelegation,
			Data:         insert,
		},
		{
			DatabaseExec: deleteUnbondingDelegation,
			Data:         deleteEntries,
		},
	}
}

func (b *unbondingDelegationsProcessor) OwnsKey(key []byte) bool {
	return bytes.HasPrefix(key, types.UnbondingDelegationKey)
}

func (b *unbondingDelegationsProcessor) Process(data tracelistener.TraceOperation) error {
	if data.Operation == tracelistener.DeleteOp.String() {
		if len(data.Key) < 41 { // 20 bytes by address, 1 prefix = 2*20 + 1
			return nil // found probably liquidity stuff being deleted
		}
		delegatorAddr := hex.EncodeToString(data.Key[1:21])
		validatorAddr := hex.EncodeToString(data.Key[21:41])
		b.l.Debugw("new unbonding_delegation delete", "delegatorAddr", delegatorAddr, "validatorAddr", validatorAddr)

		b.deleteHeightCache[unbondingDelegationCacheEntry{
			validator: validatorAddr,
			delegator: delegatorAddr,
		}] = models.UnbondingDelegationRow{
			Delegator: delegatorAddr,
			Validator: validatorAddr,
		}

		return nil
	}

	unbondingDelegation := types.UnbondingDelegation{}

	if err := p.cdc.Unmarshal(data.Value, &unbondingDelegation); err != nil {
		return err
	}

	delegator, err := b32Hex(unbondingDelegation.DelegatorAddress)
	if err != nil {
		return fmt.Errorf("cannot convert delegator address from bech32 to hex, %w", err)
	}

	validator, err := b32Hex(unbondingDelegation.ValidatorAddress)
	if err != nil {
		return fmt.Errorf("cannot convert validator address from bech32 to hex, %w", err)
	}

	entries, err := json.Marshal(unbondingDelegation.Entries)

	if err != nil {
		return fmt.Errorf("cannot convert unbonding delegation entries to string")
	}
	b.l.Debugw("new unbondingDelegation write",
		"operation", data.Operation,
		"delegator", delegator,
		"validator", validator,
		"entries", string(entries),
		"height", data.BlockHeight,
		"txHash", data.TxHash,
	)

	var entriesStore models.UnbondingDelegationEntries

	err = json.Unmarshal(entries, &entriesStore)

	if err != nil {
		return fmt.Errorf("unable to unmarshal unbonding delegation entries")
	}

	b.insertHeightCache[unbondingDelegationCacheEntry{
		validator: validator,
		delegator: delegator,
	}] = models.UnbondingDelegationRow{
		Delegator: delegator,
		Validator: validator,
		Entries:   entriesStore,
	}

	return nil
}
