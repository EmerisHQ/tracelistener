//go:build sdk_v42

package datamarshaler

import (
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank/types"
	transferTypes "github.com/cosmos/cosmos-sdk/x/ibc/applications/transfer/types"
	host "github.com/cosmos/cosmos-sdk/x/ibc/core/24-host"
	stakingTypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

var (
	BankKey                = types.BalancesPrefix
	AuthKey                = authTypes.AddressStoreKeyPrefix
	DelegationKey          = stakingTypes.DelegationKey
	IBCChannelKey          = host.KeyChannelEndPrefix
	IBCClientsKey          = host.KeyClientState
	IBCConnectionsKey      = host.KeyConnectionPrefix
	IBCDenomTracesKey      = transferTypes.DenomTraceKey
	UnbondingDelegationKey = stakingTypes.UnbondingDelegationKey
	ValidatorsKey          = stakingTypes.ValidatorsKey
)
