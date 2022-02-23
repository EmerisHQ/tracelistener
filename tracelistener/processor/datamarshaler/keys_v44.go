//go:build sdk_v44

package datamarshaler

import (
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingTypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	transferTypes "github.com/cosmos/ibc-go/v2/modules/apps/transfer/types"
	host "github.com/cosmos/ibc-go/v2/modules/core/24-host"
)

var (
	BankKey                           = types.BalancesPrefix
	AuthKey                           = authTypes.AddressStoreKeyPrefix
	DelegationKey                     = stakingTypes.DelegationKey
	IBCChannelKey                     = host.KeyChannelEndPrefix
	IBCClientsKey                     = host.KeyClientState
	IBCConnectionsKey                 = host.KeyConnectionPrefix
	IBCDenomTracesKey                 = transferTypes.DenomTraceKey
	UnbondingDelegationKey            = stakingTypes.UnbondingDelegationKey
	UnbondingDelegationByValidatorKey = stakingTypes.UnbondingDelegationByValIndexKey
	ValidatorsKey                     = stakingTypes.ValidatorsKey

	UnbondingDelegationKeys = [][]byte{
		UnbondingDelegationKey,
		UnbondingDelegationByValidatorKey,
	}
)
