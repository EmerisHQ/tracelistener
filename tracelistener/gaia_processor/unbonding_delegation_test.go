package gaia_processor

import (
	"testing"

	sdk_types "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	models "github.com/allinbits/demeris-backend-models/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener/config"
)

func TestUnbondingDelegationOwnsKey(t *testing.T) {
	u := unbondingDelegationsProcessor{}

	tests := []struct {
		name        string
		prefix      []byte
		key         string
		expectedErr bool
	}{
		{
			"Correct prefix- no error",
			types.UnbondingDelegationKey,
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

func TestUnbondingDelegationProcess(t *testing.T) {
	u := unbondingDelegationsProcessor{}

	DataProcessor, err := New(zap.NewNop().Sugar(), &config.Config{})
	require.NoError(t, err)

	gp := DataProcessor.(*Processor)
	require.NotNil(t, gp)
	p.cdc = gp.cdc

	tests := []struct {
		name                string
		unbondingDelegation types.UnbondingDelegation
		newMessage          tracelistener.TraceOperation
		expectedEr          bool
		expectedLen         int
	}{
		{
			"Delete unbonding delegation operation - no error",
			types.UnbondingDelegation{
				DelegatorAddress: "cosmos1xrnner9s783446yz3hhshpr5fpz6wzcwkvwv5j",
				ValidatorAddress: "cosmosvaloper19xawgvgn887e9gef5vkzkemwh33mtgwa6haa7s",
			},
			tracelistener.TraceOperation{
				Operation:   string(tracelistener.DeleteOp),
				Key:         []byte("QXRkbFY4cUQ2bzZKMnNoc2o5YWNwSSs5T3BkL2U1dVRxWklpN05LNWkzeTk="),
				Value:       []byte("Ci1jb3Ntb3MxeHJubmVyOXM3ODM0NDZ5ejNoaHNocHI1ZnB6Nnd6Y3drdnd2NWoSNGNvc21vc3ZhbG9wZXIxOXhhd2d2Z244ODdlOWdlZjV2a3prZW13aDMzbXRnd2E2aGFhN3MaHAiYIBILCICSuMOY/v///wEaBDEwMDAiBDExMDA="),
				BlockHeight: 0,
			},
			false,
			1,
		},
		{
			"Write unbonding delegation - no error",
			types.UnbondingDelegation{
				DelegatorAddress: "cosmos1xrnner9s783446yz3hhshpr5fpz6wzcwkvwv5j",
				ValidatorAddress: "cosmosvaloper19xawgvgn887e9gef5vkzkemwh33mtgwa6haa7s",
				Entries: []types.UnbondingDelegationEntry{
					{
						CreationHeight: 4120,
						InitialBalance: sdk_types.NewInt(1000),
						Balance:        sdk_types.NewInt(1100),
					},
				},
			},
			tracelistener.TraceOperation{
				Operation:   string(tracelistener.WriteOp),
				Key:         []byte("AtdlV8qD6o6J2shsj9acpI+9Opd/e5uTqZIi7NK5i3y9"),
				BlockHeight: 1,
				TxHash:      "066050E449C3450F943FC6227F155C19EF5C14653F268E9BAFEFE93DF9B3EDAD",
			},
			false,
			1,
		},
		{
			"Invalid addresses - error",
			types.UnbondingDelegation{},
			tracelistener.TraceOperation{
				Operation:   string(tracelistener.WriteOp),
				Key:         []byte("AtdlV8qD6o6J2shsj9acpI+9Opd/e5uTqZIi7NK5i3y9"),
				BlockHeight: 1,
				TxHash:      "A5CF62609D62ADDE56816681B6191F5F0252D2800FC2C312EB91D962AB7A97CB",
			},
			true,
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u.insertHeightCache = map[unbondingDelegationCacheEntry]models.UnbondingDelegationRow{}
			u.deleteHeightCache = map[unbondingDelegationCacheEntry]models.UnbondingDelegationRow{}
			u.l = zap.NewNop().Sugar()

			delValue, err := p.cdc.MarshalBinaryBare(&tt.unbondingDelegation)
			require.NoError(t, err)
			tt.newMessage.Value = delValue

			err = u.Process(tt.newMessage)
			if tt.expectedEr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tt.newMessage.Operation == tracelistener.DeleteOp.String() {
				require.Len(t, u.deleteHeightCache, tt.expectedLen)

				for k := range u.deleteHeightCache {
					row := u.deleteHeightCache[unbondingDelegationCacheEntry{delegator: k.delegator, validator: k.validator}]
					require.NotNil(t, row)

					return
				}
			} else {
				require.Len(t, u.insertHeightCache, tt.expectedLen)

				for k := range u.insertHeightCache {
					row := u.insertHeightCache[unbondingDelegationCacheEntry{delegator: k.delegator, validator: k.validator}]
					require.NotNil(t, row)

					delegator := row.Delegator
					delegatorAddr, err := b32Hex(tt.unbondingDelegation.DelegatorAddress)
					require.NoError(t, err)

					require.Equal(t, delegatorAddr, delegator)

					return
				}
			}
		})
	}
}

func TestUnbondingDelegationFlushCache(t *testing.T) {
	ud := unbondingDelegationsProcessor{}

	tests := []struct {
		name        string
		row         models.UnbondingDelegationRow
		isNil       bool
		expectedNil bool
	}{
		{
			"Non empty data - No error",
			models.UnbondingDelegationRow{
				Delegator: "cosmos1xrnner9s783446yz3hhshpr5fpz6wzcwkvwv5j",
				Validator: "cosmosvaloper19xawgvgn887e9gef5vkzkemwh33mtgwa6haa7s",
			},
			false,
			false,
		},
		{
			"Empty data - error",
			models.UnbondingDelegationRow{},
			false,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ud.insertHeightCache = map[unbondingDelegationCacheEntry]models.UnbondingDelegationRow{}
			ud.deleteHeightCache = map[unbondingDelegationCacheEntry]models.UnbondingDelegationRow{}

			if !tt.isNil {
				ud.insertHeightCache[unbondingDelegationCacheEntry{
					delegator: tt.row.Delegator,
					validator: tt.row.Validator,
				}] = tt.row

				ud.deleteHeightCache[unbondingDelegationCacheEntry{
					delegator: tt.row.Delegator,
					validator: tt.row.Validator,
				}] = tt.row
			}

			wop := ud.FlushCache()
			if tt.expectedNil {
				if len(wop) != 0 {
					require.Empty(t, wop[0].Data)
				}
			} else {
				require.NotNil(t, wop)
			}
		})
	}
}
