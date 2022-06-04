//go:build sdk_v44

package datamarshaler

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingTypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	transferTypes "github.com/cosmos/ibc-go/v2/modules/apps/transfer/types"
	host "github.com/cosmos/ibc-go/v2/modules/core/24-host"
)

var (
	BankKey                           = banktypes.BalancesPrefix
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

func isBankBalanceKey(key []byte) bool {
	_, _, err := splitBalanceKey(key)
	return err == nil
}

func isCW20BalanceKey(key []byte) bool {
	_, _, err := splitCW20BalanceKey(key)
	return err == nil
}

func isCW20TokenInfoKey(key []byte) bool {
	_, err := splitCW20TokenInfoKey(key)
	return err == nil
}

// splitBalanceKey returns the address and the denom of a given balance key,
// or an error if it's not valid.
// param <key> is a list of bytes. The key is a concatenation of 4 parts:
// 1. Prefix                  length: 1 Byte (always 03 for BalanceStorePrefix)
// 2. Address Len         		length: 1 Byte
// 3. Address    		          length: From 2
// 4. Denom                   length: 3 to 128 Bytes according to types.ValidateDenom
// key : <prefix><addr-len><addr><denom>
// Len	    1	       1     0-255  3-128
func splitBalanceKey(key []byte) (string, string, error) {
	const (
		// At-least: 1+1+0+3
		minLen = 1 + 1 + 3
		// At-maxLen  : 1+1+255+128
		maxLen = 1 + 1 + 255 + 128
	)
	if len(key) < minLen || len(key) > maxLen {
		return "", "",
			fmt.Errorf("malformed balance key: length %d not in range %d-%d",
				len(key), minLen, maxLen)
	}
	if !bytes.HasPrefix(key, BankKey) {
		return "", "", fmt.Errorf("not a balance store key")
	}
	// read addr
	bz, err := banktypes.AddressFromBalancesStore(key[1:])
	if err != nil {
		return "", "", fmt.Errorf("balance addr: %w", err)
	}
	addr := hex.EncodeToString(bz)
	// read denom
	denom := string(key[2+len(bz):])
	if err := types.ValidateDenom(denom); err != nil {
		return "", "", fmt.Errorf("balance denom: %w", err)
	}
	return addr, denom, nil
}

// splitDelegationKey given a key, split it into prefix, delegator and
// validator address.
// param <key> is a list of byte. The key is a concatenation of 5 parts,
// 1. Prefix                    length: 1 Byte
// 2. Delegator Address Len		length: 1 Byte
// 3. Delegator Address      	length: From 2
// 4. Validator Address Len   	length: 1 Byte
// 5. Validator Address         length: From 4
// key : <prefix><del-addr-len><del-addr><val-addr-len><val-addr>
// Len	    1	       1          0-255        1         0-255
//
// Note: Address len 0 does not make sense, but since in the SDK it's "possible" to
// have 0 len address for delegator/validator, we also consider empty address valid.
func splitDelegationKey(key []byte) (string, string, error) {
	// At-least: 3 bytes   -> 1, 2, 4 must be present. 1 byte each.
	// At-max  : 513 bytes -> 3 bytes from (1, 2, 4) + 510 bytes from (3, 5).
	if len(key) < 3 || len(key) > (1+1+255+1+255) {
		return "", "", fmt.Errorf("malformed key: length %d not in range", len(key))
	}
	_, addresses := key[0], key[1:] // Strip the prefix byte.
	delAddrLen := addresses[0]      // Gets delegator address length

	// We have to keep this check here because this function must split the two addresses,
	// FromLengthPrefix only does parsing of a well-formed length-prefix byte slice.
	if len(addresses) < int(delAddrLen) {
		return "", "", fmt.Errorf("delegator address should be %d bytes long, but it only has %d", delAddrLen, len(addresses))
	}

	totalPrefixedFirstAddressSz := delAddrLen + 1 // we are subslicing including the length-prefix, since FromLengthPrefix uses it
	delAddrBytes, err := fromLengthPrefix(addresses[:totalPrefixedFirstAddressSz])
	if err != nil {
		return "", "", fmt.Errorf("cannot parse delegator address, %w", err)
	}

	delAddr := hex.EncodeToString(delAddrBytes)

	addresses = addresses[totalPrefixedFirstAddressSz:] // Subslice past the delegator address

	valAddrBytes, err := fromLengthPrefix(addresses) // We don't do any subslicing here because FromLengthPrefix will take care of parsing errors
	if err != nil {
		return "", "", fmt.Errorf("cannot parse validator address, %w", err)
	}

	valAddr := hex.EncodeToString(valAddrBytes)
	return delAddr, valAddr, nil
}

var (
	wasmContractStorePrefix  = []byte{0x03}
	wasmContractBalanceKey   = append([]byte{0, 7}, []byte("balance")...)
	wasmContractTokenInfoKey = []byte("token_info")
)

// splitCW20BalanceKey returns the contract and the holder address of a given
// CW20 balance key, or an error if it's not valid.
// param <key> is a list of bytes. The key is a concatenation of 5 parts,
// 1. Prefix                  length: 1 Byte (always 03 for ContractStorePrefix)
// 2. Contract Address    		length: 32 Bytes
// 3. Type Len              	length: 2 Bytes
// 4. Type                  	length: From 3
// 5. Holder Address          length: the remaining bytes
// key : <prefix><contract-address><type-len><type><holder-address>
// Len	    1	          32             2     0-510    at least 43
//
// Note: Address len 0 does not make sense, but since in the SDK it's "possible" to
// have 0 len address for delegator/validator, we also consider empty address valid.
func splitCW20BalanceKey(key []byte) (string, string, error) {
	const (
		// At-least: 1+32+2+43 bytes
		minLen = 1 + 32 + 2 + 43
		// At-maxLen  : 1+32+2+510+43 bytes
		maxLen = minLen + 510
	)
	if len(key) < minLen || len(key) > maxLen {
		return "", "",
			fmt.Errorf("malformed cw20 balance key: length %d not in range %d-%d",
				len(key), minLen, maxLen)
	}
	if !bytes.HasPrefix(key, wasmContractStorePrefix) {
		return "", "", fmt.Errorf("not a wasm contract store key")
	}
	if !bytes.HasPrefix(key[33:], wasmContractBalanceKey) {
		return "", "", fmt.Errorf("not a cw20 balance key")
	}
	contractAddr := hex.EncodeToString(key[1:33])
	// holder addr must be bech32 decoded
	_, bz, err := bech32.DecodeAndConvert(string(key[42:]))
	if err != nil {
		return "", "", fmt.Errorf("decode holder address: %w", err)
	}
	holderAddr := hex.EncodeToString(bz)
	return contractAddr, holderAddr, nil
}

// splitCW20TokenInfoKey returns the contract address of a given
// CW20 token_info key, or an error if it's not valid.
// param <key> is a list of bytes. The key is a concatenation of 5 parts,
// 1. Prefix                  length: 1 Byte (always 03 for ContractStorePrefix)
// 2. Contract Address    		length: 32 Bytes
// 4. Type                  	length: 10 Bytes (always token_info)
// key : <prefix><contract-address><type>
// Len	    1	          32           10
func splitCW20TokenInfoKey(key []byte) (string, error) {
	const expectedLen = 1 + 32 + 10
	if len(key) != expectedLen {
		return "", fmt.Errorf(
			"malformed cw20 token_info key: length %d not equal to %d",
			len(key), expectedLen,
		)
	}
	if !bytes.HasPrefix(key, wasmContractStorePrefix) {
		return "", fmt.Errorf("not a wasm contract store key")
	}
	if !bytes.HasPrefix(key[33:], wasmContractTokenInfoKey) {
		return "", fmt.Errorf("not a cw20 token_info key")
	}
	contractAddr := hex.EncodeToString(key[1:33])
	return contractAddr, nil
}
