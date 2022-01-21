//go:build sdk_v42

package datamarshaler

import (
	"time"

	ics23 "github.com/confio/ics23/go"
	"github.com/cosmos/cosmos-sdk/codec"
	codecTypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	transferTypes "github.com/cosmos/cosmos-sdk/x/ibc/applications/transfer/types"
	clientTypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	connectionTypes "github.com/cosmos/cosmos-sdk/x/ibc/core/03-connection/types"
	ibcChannelTypes "github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"
	ibcTypes "github.com/cosmos/cosmos-sdk/x/ibc/core/23-commitment/types"
	lightClientTypes "github.com/cosmos/cosmos-sdk/x/ibc/light-clients/07-tendermint/types"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
	stakingTypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/gogo/protobuf/proto"
)

func marshalIfaceOrPanic(p proto.Message) []byte {
	data, err := getCodec().MarshalInterface(p)
	if err != nil {
		panic(err)
	}

	return data
}

func marshalOrPanic(p codec.ProtoMarshaler) []byte {
	return getCodec().MustMarshalBinaryBare(p)
}

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

func (d TestDataMarshaler) IBCChannel(state, ordering int32, counterPortID, counterChannelID, hop string) []byte {
	c := ibcChannelTypes.Channel{
		State:    ibcChannelTypes.State(state),
		Ordering: ibcChannelTypes.Order(ordering),
		Counterparty: ibcChannelTypes.Counterparty{
			PortId:    counterPortID,
			ChannelId: counterChannelID,
		},
		ConnectionHops: []string{hop},
	}

	return marshalOrPanic(&c)
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

func (d TestDataMarshaler) IBCClient(state TestClientState) []byte {
	c := lightClientTypes.ClientState{
		ChainId: state.ChainId,
		TrustLevel: lightClientTypes.Fraction{
			Numerator:   state.TrustLevel.Numerator,
			Denominator: state.TrustLevel.Denominator,
		},
		TrustingPeriod:  state.TrustingPeriod,
		UnbondingPeriod: state.UnbondingPeriod,
		MaxClockDrift:   state.MaxClockDrift,
		FrozenHeight: clientTypes.Height{
			RevisionNumber: state.FrozenHeight.Number,
			RevisionHeight: state.FrozenHeight.Height,
		},
		LatestHeight: clientTypes.NewHeight(state.LatestHeight.Height, state.LatestHeight.Number),
	}

	for _, ps := range state.ProofSpecs {
		c.ProofSpecs = append(c.ProofSpecs, &ics23.ProofSpec{
			LeafSpec: &ics23.LeafOp{
				Hash:   ics23.HashOp(ps.Hash),
				Length: ics23.LengthOp(ps.Length),
			},
		})
	}

	return marshalIfaceOrPanic(&c)
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

func (d TestDataMarshaler) IBCConnection(conn TestConnection) []byte {
	c := connectionTypes.ConnectionEnd{
		ClientId: conn.ClientId,
		Versions: []*connectionTypes.Version{
			{
				Identifier: conn.VersionIdentifier,
			},
		},
		State: connectionTypes.State(conn.State),
		Counterparty: connectionTypes.Counterparty{
			ClientId:     conn.CountClientID,
			ConnectionId: conn.CountConnectionID,
			Prefix: ibcTypes.MerklePrefix{
				KeyPrefix: []byte(conn.CountPrefix),
			},
		},
		DelayPeriod: conn.DelayPeriod,
	}

	return marshalOrPanic(&c)
}

func (d TestDataMarshaler) MapConnectionState(s int32) string {
	return connectionTypes.State_name[s]
}

func (d TestDataMarshaler) IBCDenomTraces(path, baseDenom string) []byte {
	t := transferTypes.DenomTrace{
		Path:      path,
		BaseDenom: baseDenom,
	}

	return marshalOrPanic(&t)
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
