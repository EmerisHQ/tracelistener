package tracelistener

import (
	"encoding/hex"
	"fmt"
)

// SplitDelegatorValidatorAddress given a key split it into delegator and
// validator address.
// param <key> is a list of byte where the key is a concatenation of 5 parts
// 1. Prefix                       1 Byte
// 2. Delegator Address length     1 Byte
// 3. Delegator Address            From 2
// 4. Validator Address Length +   1 Byte
// 5. Validator Address            From 4
func SplitDelegatorValidatorAddress(key []byte) (string, string, error) {
	if len(key) < 3 { // At least 1, 2, 4 must be present
		return "", "", fmt.Errorf("malformed key: %v", key)
	}

	addresses := key[1:] // Strip the prefix byte.
	delegatorAddrLength := addresses[0]
	addresses = addresses[1:] // Strip the address byte.
	delegatorAddr := hex.EncodeToString(addresses[0:delegatorAddrLength])

	addresses = addresses[delegatorAddrLength:] // Strip the delegator address portion.
	addresses = addresses[1:]                   // Strip the address length byte.

	validatorAddr := hex.EncodeToString(addresses[0:])
	return delegatorAddr, validatorAddr, nil
}
