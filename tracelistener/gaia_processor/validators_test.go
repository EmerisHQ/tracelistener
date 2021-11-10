package gaia_processor

import (
	"testing"

	types1 "github.com/cosmos/cosmos-sdk/codec/types"
	sdk_types "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
	gaia "github.com/cosmos/gaia/v4/app"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	models "github.com/allinbits/demeris-backend-models/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener"
)

func TestValidatorProcess(t *testing.T) {
	tests := []struct {
		name       string
		validator  types.Validator
		newMessage tracelistener.TraceOperation
		wantErr    bool
	}{
		{
			"Delete validator operation - no error",
			types.Validator{
				OperatorAddress: "cosmosvaloper19xawgvgn887e9gef5vkzkemwh33mtgwa6haa7s",
				ConsensusPubkey: &types1.Any{
					Value: []byte("dlxLxyNmux++E2mjN4GR6u/whv8uMsMTIS1Tw1WylJw="),
				},
				Jailed:          false,
				Status:          types.Bonded,
				Tokens:          sdk_types.NewInt(90000030000),
				DelegatorShares: sdk_types.NewDec(90000030000),
				UnbondingHeight: 0,
				Commission: types.Commission{
					CommissionRates: types.CommissionRates{
						Rate:          sdk_types.NewDec(100),
						MaxRate:       sdk_types.NewDec(200),
						MaxChangeRate: sdk_types.NewDec(1000),
					},
				},
				MinSelfDelegation: sdk_types.NewInt(1),
			},
			tracelistener.TraceOperation{
				Operation: string(tracelistener.DeleteOp),
				Key:       []byte("cosmosvaloper19xawgvgn887e9gef5vkzkemwh33mtgwa6haa7s"),
			},
			false,
		},
		{
			"Write validator operation - no error",
			types.Validator{
				OperatorAddress: "cosmosvaloper19xawgvgn887e9gef5vkzkemwh33mtgwa6haa7s",
				ConsensusPubkey: &types1.Any{
					Value: []byte("dlxLxyNmux++E2mjN4GR6u/whv8uMsMTIS1Tw1WylJw="),
				},
				Jailed:          false,
				Status:          types.Bonded,
				Tokens:          sdk_types.NewInt(90000030000),
				DelegatorShares: sdk_types.NewDec(90000030000),
				UnbondingHeight: 0,
				Commission: types.Commission{
					CommissionRates: types.CommissionRates{
						Rate:          sdk_types.NewDec(100),
						MaxRate:       sdk_types.NewDec(200),
						MaxChangeRate: sdk_types.NewDec(1000),
					},
				},
				MinSelfDelegation: sdk_types.NewInt(1),
			},
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := validatorsProcessor{}
			v.insertValidatorsCache = map[validatorCacheEntry]models.ValidatorRow{}
			v.deleteValidatorsCache = map[validatorCacheEntry]models.ValidatorRow{}
			v.l = zap.NewNop().Sugar()

			cdc, _ := gaia.MakeCodecs()

			delValue, _ := cdc.MarshalBinaryBare(&tt.validator)
			tt.newMessage.Value = delValue

			err := v.Process(tt.newMessage)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

		})
	}
}
