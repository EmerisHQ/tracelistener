package gaia_processor

import (
	"bytes"
	"encoding/hex"

	"github.com/allinbits/demeris-backend/models"

	"github.com/allinbits/demeris-backend/tracelistener"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
	"go.uber.org/zap"
)

type validatorCacheEntry struct {
	operator string
}
type validatorsProcessor struct {
	l                     *zap.SugaredLogger
	insertValidatorsCache map[validatorCacheEntry]models.ValidatorRow
	deleteValidatorsCache map[validatorCacheEntry]models.ValidatorRow
}

func (*validatorsProcessor) TableSchema() string {
	return createValidatorsTable
}

func (p *validatorsProcessor) ModuleName() string {
	return "validators"
}

func (p *validatorsProcessor) FlushCache() []tracelistener.WritebackOp {

	if len(p.insertValidatorsCache) == 0 && len(p.deleteValidatorsCache) == 0 {
		return nil
	}

	insertValidators := make([]models.DatabaseEntrier, 0, len(p.insertValidatorsCache))
	deleteValidators := make([]models.DatabaseEntrier, 0, len(p.deleteValidatorsCache))

	if len(p.insertValidatorsCache) != 0 {
		for _, v := range p.insertValidatorsCache {
			insertValidators = append(insertValidators, v)
		}
	}

	p.insertValidatorsCache = map[validatorCacheEntry]models.ValidatorRow{}

	if len(p.deleteValidatorsCache) != 0 {
		for _, v := range p.deleteValidatorsCache {
			deleteValidators = append(deleteValidators, v)
		}
	}

	p.deleteValidatorsCache = map[validatorCacheEntry]models.ValidatorRow{}

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
	return bytes.HasPrefix(key, types.ValidatorsKey)
}

func (b *validatorsProcessor) Process(data tracelistener.TraceOperation) error {

	if data.Operation == tracelistener.DeleteOp.String() {
		if len(data.Key) < 21 {
			return nil
		}

		operatorAddress := hex.EncodeToString(data.Key[1:21])
		b.l.Debugw("new validator delete", "operator address", operatorAddress)

		b.deleteValidatorsCache[validatorCacheEntry{
			operator: operatorAddress,
		}] = models.ValidatorRow{
			OperatorAddress: operatorAddress,
		}

		return nil
	}

	v := types.Validator{}

	if err := p.cdc.UnmarshalBinaryBare(data.Value, &v); err != nil {
		return err
	}

	val := string(v.ConsensusPubkey.GetValue())

	k := hex.EncodeToString(data.Key)

	b.l.Debugw("new validator write",
		"operator_address", v.OperatorAddress,
		"height", data.BlockHeight,
		"txHash", data.TxHash,
		"cons pub key type", data.TxHash,
		"cons pub key", val,
		"key", k,
	)

	b.insertValidatorsCache[validatorCacheEntry{
		operator: v.OperatorAddress,
	}] = models.ValidatorRow{
		OperatorAddress:      v.OperatorAddress,
		ConsensusPubKeyType:  v.ConsensusPubkey.GetTypeUrl(),
		ConsensusPubKeyValue: v.ConsensusPubkey.Value,
		Jailed:               v.Jailed,
		Status:               int32(v.Status),
		Tokens:               v.Tokens.String(),
		DelegatorShares:      v.DelegatorShares.String(),
		Moniker:              v.Description.Moniker,
		Identity:             v.Description.Identity,
		Website:              v.Description.Website,
		SecurityContact:      v.Description.SecurityContact,
		Details:              v.Description.Details,
		UnbondingHeight:      v.UnbondingHeight,
		UnbondingTime:        v.UnbondingTime.String(),
		CommissionRate:       v.Commission.CommissionRates.Rate.String(),
		MaxRate:              v.Commission.CommissionRates.MaxRate.String(),
		MaxChangeRate:        v.Commission.CommissionRates.MaxChangeRate.String(),
		UpdateTime:           v.Commission.UpdateTime.String(),
		MinSelfDelegation:    v.MinSelfDelegation.String(),
	}

	return nil
}
