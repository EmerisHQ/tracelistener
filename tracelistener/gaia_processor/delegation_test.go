package gaia_processor

import (
	"testing"

	sdk_types "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
	gaia "github.com/cosmos/gaia/v4/app"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	models "github.com/allinbits/demeris-backend-models/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener"
)

func TestDelegationProcess(t *testing.T) {
	tests := []struct {
		name       string
		delegation types.Delegation
		newMessage tracelistener.TraceOperation
		wantErr    bool
	}{
		{
			"no error when queueing delete delegation message accepted by the processor",
			types.Delegation{
				DelegatorAddress: "cosmos1xrnner9s783446yz3hhshpr5fpz6wzcwkvwv5j",
				ValidatorAddress: "cosmosvaloper19xawgvgn887e9gef5vkzkemwh33mtgwa6haa7s",
				Shares:           sdk_types.NewDec(100),
			},
			tracelistener.TraceOperation{
				Operation:   string(tracelistener.DeleteOp),
				Key:         []byte("QXRkbFY4cUQ2bzZKMnNoc2o5YWNwSSs5T3BkL2U1dVRxWklpN05LNWkzeTk="),
				Value:       []byte("100stake"),
				BlockHeight: 0,
			},
			false,
		},
		{
			"no error when queueing new message accepted by the processor",
			types.Delegation{
				DelegatorAddress: "cosmos1xrnner9s783446yz3hhshpr5fpz6wzcwkvwv5j",
				ValidatorAddress: "cosmosvaloper19xawgvgn887e9gef5vkzkemwh33mtgwa6haa7s",
				Shares:           sdk_types.NewDec(100),
			},
			tracelistener.TraceOperation{
				Operation:   string(tracelistener.WriteOp),
				Key:         []byte("AtdlV8qD6o6J2shsj9acpI+9Opd/e5uTqZIi7NK5i3y9"),
				BlockHeight: 1,
				TxHash:      "A5CF62609D62ADDE56816681B6191F5F0252D2800FC2C312EB91D962AB7A97CB",
			},
			false,
		},
		{
			"error while writing delegation - eof",
			types.Delegation{
				DelegatorAddress: "",
				ValidatorAddress: "",
				Shares:           sdk_types.NewDec(100),
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
			d := delegationsProcessor{}
			d.insertHeightCache = map[delegationCacheEntry]models.DelegationRow{}
			d.deleteHeightCache = map[delegationCacheEntry]models.DelegationRow{}
			d.l = zap.NewNop().Sugar()

			cdc, _ := gaia.MakeCodecs()

			delValue, _ := cdc.MarshalBinaryBare(&tt.delegation)
			tt.newMessage.Value = delValue

			err := d.Process(tt.newMessage)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

		})
	}
}
