package processor

import (
	"encoding/hex"
	"fmt"

	"github.com/cosmos/btcutil/bech32"
)

// for some weird reason golangci-lint hates us, and marks this func as unused, weird
//nolint:unused,deadcode
func b32Hex(s string) (string, error) {
	_, b, err := decodeAndConvert(s)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

// vendored from cosmos-sdk, no need to bring it as a dep here just because of this
// for some weird reason golangci-lint hates us, and marks this func as unused, weird
//nolint:unused,deadcode
func decodeAndConvert(bech string) (string, []byte, error) {
	hrp, data, err := bech32.Decode(bech, 1023)
	if err != nil {
		return "", nil, fmt.Errorf("decoding bech32 failed: %w", err)
	}

	converted, err := bech32.ConvertBits(data, 5, 8, false)
	if err != nil {
		return "", nil, fmt.Errorf("decoding bech32 failed: %w", err)
	}

	return hrp, converted, nil
}
