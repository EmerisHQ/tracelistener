//go:build sdk_v42

package datamarshaler

import (
	"bytes"

	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	transferTypes "github.com/cosmos/cosmos-sdk/x/ibc/applications/transfer/types"
	host "github.com/cosmos/cosmos-sdk/x/ibc/core/24-host"
	stakingTypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

var (
	BankKey                = bankTypes.BalancesPrefix
	AuthKey                = authTypes.AddressStoreKeyPrefix
	DelegationKey          = stakingTypes.DelegationKey
	IBCChannelKey          = host.KeyChannelEndPrefix
	IBCClientsKey          = host.KeyClientState
	IBCConnectionsKey      = host.KeyConnectionPrefix
	IBCDenomTracesKey      = transferTypes.DenomTraceKey
	UnbondingDelegationKey = stakingTypes.UnbondingDelegationKey
	ValidatorsKey          = stakingTypes.ValidatorsKey

	UnbondingDelegationKeys = [][]byte{UnbondingDelegationKey}
)

func isBankBalanceKey(key []byte) bool {
	return bytes.HasPrefix(key, BankKey)
}

func isCW20BalanceKey(key []byte) bool {
	// CW20Balance not implemented in v42
  return false
}

func isCW20TokenInfoKey(key []byte) bool {
	// CW20TokenInfo not implemented in v42
  return false
}
