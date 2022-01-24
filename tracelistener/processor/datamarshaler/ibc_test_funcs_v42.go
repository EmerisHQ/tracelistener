//go:build sdk_v42

package datamarshaler

import (
	ics23 "github.com/confio/ics23/go"
	transferTypes "github.com/cosmos/cosmos-sdk/x/ibc/applications/transfer/types"
	clientTypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	connectionTypes "github.com/cosmos/cosmos-sdk/x/ibc/core/03-connection/types"
	ibcChannelTypes "github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"
	ibcTypes "github.com/cosmos/cosmos-sdk/x/ibc/core/23-commitment/types"
	lightClientTypes "github.com/cosmos/cosmos-sdk/x/ibc/light-clients/07-tendermint/types"
)

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

func (d TestDataMarshaler) IBCDenomTraces(path, baseDenom string) []byte {
	t := transferTypes.DenomTrace{
		Path:      path,
		BaseDenom: baseDenom,
	}

	return marshalOrPanic(&t)
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
