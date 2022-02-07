package processor

import (
	"encoding/hex"
	"fmt"

	"github.com/cosmos/btcutil/bech32"
)

func b32Hex(s string) (string, error) {
	_, b, err := decodeAndConvert(s)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

// vendored from cosmos-sdk, no need to bring it as a dep here just because of this
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
