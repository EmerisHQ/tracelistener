package datamarshaler

import (
	"time"

	codecTypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
	stakingTypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (d TestDataMarshaler) Account(accountNumber, sequenceNumber uint64, address string) []byte {
	a := authtypes.BaseAccount{
		Address:       address,
		AccountNumber: accountNumber,
		Sequence:      sequenceNumber,
	}

	return marshalIfaceOrPanic(&a)
}

func (d TestDataMarshaler) Coin(denom string, amount int64) []byte {
	c := sdk.Coin{
		Denom:  denom,
		Amount: sdk.NewInt(amount),
	}

	return marshalOrPanic(&c)
}

func (d TestDataMarshaler) Delegation(validator, delegator string, shares int64) []byte {
	del := stakingTypes.Delegation{
		ValidatorAddress: validator,
		DelegatorAddress: delegator,
		Shares:           sdk.NewDec(shares),
	}

	return marshalOrPanic(&del)
}

// What follows are type definitions to aid IBC Client marshaling function.
// Having all those fields as a func parameter hurts my brain, so I decided
// to build structs instead.
// Freely inspired by the IBC Go package :-)
type TestFraction struct {
	Numerator   uint64
	Denominator uint64
}

type TestProofSpec struct {
	Hash   int32
	Length int32
}

type TestHeight struct {
	Number uint64
	Height uint64
}

type TestClientState struct {
	ChainId                      string
	TrustLevel                   TestFraction
	TrustingPeriod               time.Duration
	UnbondingPeriod              time.Duration
	MaxClockDrift                time.Duration
	FrozenHeight                 TestHeight
	LatestHeight                 TestHeight
	ProofSpecs                   []TestProofSpec
	UpgradePath                  []string
	AllowUpdateAfterExpiry       bool
	AllowUpdateAfterMisbehaviour bool
}

type TestConnection struct {
	ClientId          string
	VersionIdentifier string
	State             int32
	CountClientID     string
	CountConnectionID string
	CountPrefix       string
	DelayPeriod       uint64
}

type TestValCommission struct {
	Rate          int64
	MaxRate       int64
	MaxChangeRate int64
}
type TestValidator struct {
	OperatorAddress   string
	ConsensusPubkey   string
	Jailed            bool
	Status            int32
	Tokens            int64
	DelegatorShares   int64
	UnbondingHeight   int64
	UnbondingTime     time.Time
	Commission        TestValCommission
	MinSelfDelegation int64
}

func (d TestDataMarshaler) Validator(v TestValidator) []byte {
	vv := stakingTypes.Validator{
		OperatorAddress: v.OperatorAddress,
		ConsensusPubkey: &codecTypes.Any{
			Value: []byte(v.ConsensusPubkey),
		},
		Jailed:          v.Jailed,
		Status:          stakingTypes.BondStatus(v.Status),
		Tokens:          sdk.NewInt(v.Tokens),
		DelegatorShares: sdk.NewDec(v.DelegatorShares),
		UnbondingHeight: v.UnbondingHeight,
		Commission: types.Commission{
			CommissionRates: types.CommissionRates{
				Rate:          sdk.NewDec(v.Commission.Rate),
				MaxRate:       sdk.NewDec(v.Commission.MaxRate),
				MaxChangeRate: sdk.NewDec(v.Commission.MaxChangeRate),
			},
		},
		MinSelfDelegation: sdk.NewInt(v.MinSelfDelegation),
	}

	return marshalOrPanic(&vv)
}

type TestUnbondingDelegationEntry struct {
	Height         int64
	Completion     time.Time
	InitialBalance int64
	Balance        int64
}

type TestUnbondingDelegation struct {
	Delegator string
	Validator string
	Entries   []TestUnbondingDelegationEntry
}

func (d TestDataMarshaler) UnbondingDelegation(u TestUnbondingDelegation) []byte {
	uu := types.UnbondingDelegation{
		DelegatorAddress: u.Delegator,
		ValidatorAddress: u.Validator,
	}

	for _, e := range u.Entries {
		uu.Entries = append(uu.Entries,
			types.UnbondingDelegationEntry{
				CreationHeight: e.Height,
				InitialBalance: sdk.NewInt(e.InitialBalance),
				Balance:        sdk.NewInt(e.Balance),
				CompletionTime: e.Completion,
			},
		)
	}

	return marshalOrPanic(&uu)
}
