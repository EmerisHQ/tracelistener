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

func TestUnbondingDelegationProcess(t *testing.T) {
	DataProcessor, _ := New(zap.NewNop().Sugar(), &config.Config{})

	gp := DataProcessor.(*Processor)
	require.NotNil(t, gp)
	p.cdc = gp.cdc

	tests := []struct {
		name                string
		unbondingDelegation types.UnbondingDelegation
		newMessage          tracelistener.TraceOperation
		wantErr             bool
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
		},
		{
			"Invalid addresses - error",
			types.UnbondingDelegation{
				DelegatorAddress: "",
				ValidatorAddress: "",
			},
			tracelistener.TraceOperation{
				Operation:   string(tracelistener.WriteOp),
				Key:         []byte("AtdlV8qD6o6J2shsj9acpI+9Opd/e5uTqZIi7NK5i3y9"),
				BlockHeight: 1,
				TxHash:      "A5CF62609D62ADDE56816681B6191F5F0252D2800FC2C312EB91D962AB7A97CB",
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := unbondingDelegationsProcessor{}
			u.insertHeightCache = map[unbondingDelegationCacheEntry]models.UnbondingDelegationRow{}
			u.deleteHeightCache = map[unbondingDelegationCacheEntry]models.UnbondingDelegationRow{}
			u.l = zap.NewNop().Sugar()

			delValue, _ := p.cdc.MarshalBinaryBare(&tt.unbondingDelegation)
			tt.newMessage.Value = delValue

			err := u.Process(tt.newMessage)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

		})
	}
}
