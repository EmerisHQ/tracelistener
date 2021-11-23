package gaia_processor

import (
	"testing"

	types1 "github.com/cosmos/cosmos-sdk/codec/types"
	sdk_types "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	models "github.com/allinbits/demeris-backend-models/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener/config"
)

func TestValidatorProcessOwnsKey(t *testing.T) {
	u := validatorsProcessor{}

	tests := []struct {
		name        string
		prefix      []byte
		key         string
		expectedErr bool
	}{
		{
			"Correct prefix- no error",
			types.ValidatorsKey,
			"key",
			false,
		},
		{
			"Incorrect prefix- error",
			[]byte{0x0},
			"key",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.expectedErr {
				require.False(t, u.OwnsKey(append(tt.prefix, []byte(tt.key)...)))
			} else {
				require.True(t, u.OwnsKey(append(tt.prefix, []byte(tt.key)...)))
			}
		})
	}
}

func TestValidatorProcess(t *testing.T) {
	v := validatorsProcessor{}

	DataProcessor, err := New(zap.NewNop().Sugar(), &config.Config{})
	require.NoError(t, err)

	gp := DataProcessor.(*Processor)
	require.NotNil(t, gp)
	p.cdc = gp.cdc

	tests := []struct {
		name        string
		validator   types.Validator
		newMessage  tracelistener.TraceOperation
		expectedErr bool
		expectedLen int
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
			1,
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
			1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v.insertValidatorsCache = map[validatorCacheEntry]models.ValidatorRow{}
			v.deleteValidatorsCache = map[validatorCacheEntry]models.ValidatorRow{}
			v.l = zap.NewNop().Sugar()

			value, err := p.cdc.MarshalBinaryBare(&tt.validator)
			require.NoError(t, err)
			tt.newMessage.Value = value

			err = v.Process(tt.newMessage)
			if tt.expectedErr {
				require.Error(t, err)
				return
			} else {
				require.NoError(t, err)
			}

			if tt.newMessage.Operation == tracelistener.DeleteOp.String() {
				require.Len(t, v.deleteValidatorsCache, tt.expectedLen)

				for k, _ := range v.deleteValidatorsCache {
					row := v.deleteValidatorsCache[validatorCacheEntry{operator: k.operator}]
					require.NotNil(t, row)

					return
				}
			} else {
				require.Len(t, v.insertValidatorsCache, tt.expectedLen)

				for k, _ := range v.insertValidatorsCache {
					row := v.insertValidatorsCache[validatorCacheEntry{operator: k.operator}]
					require.NotNil(t, row)

					status := row.Status
					require.Equal(t, int32(tt.validator.Status), status)

					return
				}
			}
		})
	}
}
