package tracelistener

import (
	"encoding/hex"
	"fmt"
)

// SplitDelegationKey given a key, split it into prefix, delegator and
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
func SplitDelegationKey(key []byte) (string, string, error) {
	// At-least: 3 bytes   -> 1, 2, 4 must be present. 1 byte each.
	// At-max  : 513 bytes -> 3 bytes from (1, 2, 4) + 510 bytes from (3, 5).
	if len(key) < 3 || len(key) > (1+1+255+1+255) {
		return "", "", fmt.Errorf("malformed key: length %d not in range", len(key))
	}
	_, addresses := key[0], key[1:] // Strip the prefix byte.
	delAddrLen := addresses[0]      // Strip the delegator length byte.
	if delAddrLen > 255 {
		return "", "", fmt.Errorf("malformed key: delegator address length out of range %d", delAddrLen)
	}

	// Check if we have enough bytes for the address of delegator + at least 1 byte for validator address length
	if len(addresses) < int(delAddrLen) {
		return "", "", fmt.Errorf(
			"malformed key: delegator key length not sufficient. want atlease: %d got: %d",
			delAddrLen,
			len(addresses),
		)
	}

	totalPrefixedFirstAddressSz := delAddrLen + 1 // we are subslicing including the length-prefix, since FromLengthPrefix uses it
	delAddrBytes, err := FromLengthPrefix(addresses[:totalPrefixedFirstAddressSz])
	if err != nil {
		return "", "", err
	}

	delAddr := hex.EncodeToString(delAddrBytes)

	addresses = addresses[totalPrefixedFirstAddressSz:] // Strip the delegator address,
	valAddrLen := addresses[0]                          // Strip the address length byte.
	if valAddrLen > 255 {
		return "", "", fmt.Errorf("malformed key: validator address length out of range %d", valAddrLen)
	}
	// Check if we have exact number of bytes for the address of validator.
	// Subtracting 1 here since it's the length-prefix, which will be consumed later.
	if len(addresses)-1 != int(valAddrLen) {
		return "", "", fmt.Errorf(
			"malformed key: validator address length out of range. want: %d got: %d",
			valAddrLen,
			len(addresses)-1,
		)
	}

	totalPrefixedSecondAddressSz := valAddrLen + 1
	valAddrBytes, err := FromLengthPrefix(addresses[:totalPrefixedSecondAddressSz])
	if err != nil {
		return "", "", err
	}

	valAddr := hex.EncodeToString(valAddrBytes)
	return delAddr, valAddr, nil
}

// FromLengthPrefix returns the amount of data signaled by the single-byte length prefix in rawData.
func FromLengthPrefix(rawData []byte) ([]byte, error) {
	if rawData == nil {
		return nil, fmt.Errorf("data is nil")
	}

	length := int(rawData[0])
	rawData = rawData[1:]
	if len(rawData) != length {
		return nil, fmt.Errorf("length prefix signals %d bytes, but total data is %d bytes long", length, len(rawData))
	}

	data := rawData[:length]

	return data, nil
}
