package processor

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	models "github.com/allinbits/demeris-backend-models/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener/config"
	"github.com/allinbits/tracelistener/tracelistener/processor/datamarshaler"
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
			datamarshaler.ValidatorsKey,
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

	tests := []struct {
		name        string
		validator   datamarshaler.TestValidator
		newMessage  tracelistener.TraceOperation
		expectedErr bool
		expectedLen int
	}{
		{
			"Delete validator operation - no error",
			datamarshaler.TestValidator{
				OperatorAddress: "cosmosvaloper19xawgvgn887e9gef5vkzkemwh33mtgwa6haa7s",
				ConsensusPubkey: "dlxLxyNmux++E2mjN4GR6u/whv8uMsMTIS1Tw1WylJw=",
				Jailed:          false,
				Status:          3, // bonded
				Tokens:          90000030000,
				DelegatorShares: 90000030000,
				UnbondingHeight: 0,
				Commission: datamarshaler.TestValCommission{
					Rate:          100,
					MaxRate:       200,
					MaxChangeRate: 1000,
				},
				MinSelfDelegation: 1,
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
			datamarshaler.TestValidator{
				OperatorAddress: "cosmosvaloper19xawgvgn887e9gef5vkzkemwh33mtgwa6haa7s",
				ConsensusPubkey: "dlxLxyNmux++E2mjN4GR6u/whv8uMsMTIS1Tw1WylJw=",
				Jailed:          false,
				Status:          3, // bonded
				Tokens:          90000030000,
				DelegatorShares: 90000030000,
				UnbondingHeight: 0,
				Commission: datamarshaler.TestValCommission{
					Rate:          100,
					MaxRate:       200,
					MaxChangeRate: 1000,
				},
				MinSelfDelegation: 1,
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

			tt.newMessage.Value = datamarshaler.NewTestDataMarshaler().Validator(tt.validator)

			err = v.Process(tt.newMessage)
			if tt.expectedErr {
				require.Error(t, err)
				return
			} else {
				require.NoError(t, err)
			}

			if tt.newMessage.Operation == tracelistener.DeleteOp.String() {
				require.Len(t, v.deleteValidatorsCache, tt.expectedLen)

				for k := range v.deleteValidatorsCache {
					row := v.deleteValidatorsCache[validatorCacheEntry{operator: k.operator}]
					require.NotNil(t, row)

					return
				}
			} else {
				require.Len(t, v.insertValidatorsCache, tt.expectedLen)

				for k := range v.insertValidatorsCache {
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

func TestValidatorFlushCache(t *testing.T) {
	v := validatorsProcessor{}

	tests := []struct {
		name        string
		row         models.ValidatorRow
		isNil       bool
		expectedNil bool
	}{
		{
			"Non empty data - No error",
			models.ValidatorRow{
				OperatorAddress: "cosmosvaloper19xawgvgn887e9gef5vkzkemwh33mtgwa6haa7s",
				Jailed:          false,
			},
			false,
			false,
		},
		{
			"Empty data - error",
			models.ValidatorRow{},
			true,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v.insertValidatorsCache = map[validatorCacheEntry]models.ValidatorRow{}
			v.deleteValidatorsCache = map[validatorCacheEntry]models.ValidatorRow{}

			if !tt.isNil {
				v.insertValidatorsCache[validatorCacheEntry{
					operator: tt.row.OperatorAddress,
				}] = tt.row
				v.deleteValidatorsCache[validatorCacheEntry{
					operator: tt.row.OperatorAddress,
				}] = tt.row
			}

			wop := v.FlushCache()
			if tt.expectedNil {
				require.Nil(t, wop)
			} else {
				require.NotNil(t, wop)
			}
		})
	}
}
