package gaia_processor

import (
	"encoding/hex"

	"github.com/cosmos/cosmos-sdk/types/bech32"
)

func b32Hex(s string) (string, error) {
	_, b, err := bech32.DecodeAndConvert(s)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}
