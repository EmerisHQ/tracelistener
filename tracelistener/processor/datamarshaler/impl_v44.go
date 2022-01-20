//go:build sdk_v44

package datamarshaler

import (
	"encoding/hex"

	models "github.com/allinbits/demeris-backend-models/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank/types"
)

func (d DataMarshaler) Bank(data tracelistener.TraceOperation) (models.BalanceRow, error) {
	addrBytes := data.Key
	pLen := len(types.BalancesPrefix)
	addr := addrBytes[pLen : pLen+20]

	coins := sdk.Coin{
		Amount: sdk.NewInt(0),
	}

	if err := p.cdc.UnmarshalBinaryBare(data.Value, &coins); err != nil {
		return err
	}

	if !coins.IsValid() {
		return models.BalanceRow{}, nil
	}

	hAddr := hex.EncodeToString(addr)
	b.l.Debugw("new bank store write",
		"operation", data.Operation,
		"address", hAddr,
		"new_balance", coins.String(),
		"height", data.BlockHeight,
		"txHash", data.TxHash,
	)
	return models.BalanceRow{
		Address:     hAddr,
		Amount:      coins.String(),
		Denom:       coins.Denom,
		BlockHeight: data.BlockHeight,
	}, nil
}

func (d DataMarshaler) Auth(data tracelistener.TraceOperation) error {
	panic("not implemented") // TODO: Implement
}

func (d DataMarshaler) Delegations(data tracelistener.TraceOperation) error {
	panic("not implemented") // TODO: Implement
}

func (d DataMarshaler) IBCChannels(data tracelistener.TraceOperation) error {
	panic("not implemented") // TODO: Implement
}

func (d DataMarshaler) IBCClients(data tracelistener.TraceOperation) error {
	panic("not implemented") // TODO: Implement
}

func (d DataMarshaler) IBCConnections(data tracelistener.TraceOperation) error {
	panic("not implemented") // TODO: Implement
}

func (d DataMarshaler) IBCDenomTraces(data tracelistener.TraceOperation) error {
	panic("not implemented") // TODO: Implement
}

func (d DataMarshaler) UnbondingDelegations(data tracelistener.TraceOperation) error {
	panic("not implemented") // TODO: Implement
}

func (d DataMarshaler) Validators(data tracelistener.TraceOperation) error {
	panic("not implemented") // TODO: Implement
}
