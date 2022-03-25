//go:build sdk_v44

package datamarshaler

import (
	"testing"

	clientTypes "github.com/cosmos/ibc-go/v2/modules/core/02-client/types"
	"github.com/stretchr/testify/require"
)

func Test_ParseIBCChainID(t *testing.T) {
	tt := []struct {
		name        string
		fullChainID string
		height      clientTypes.Height
		expected    string
	}{
		{
			name:        "with revision height",
			fullChainID: "cosmoshub-4-999999",
			height:      clientTypes.NewHeight(4, 999999),
			expected:    "cosmoshub-4",
		},
		{
			name:        "without revision height",
			fullChainID: "cosmoshub-4",
			height:      clientTypes.NewHeight(4, 999999),
			expected:    "cosmoshub-4",
		},
		{
			name:        "without revision height",
			fullChainID: "cosmoshub-4",
			height:      clientTypes.NewHeight(4, 999999),
			expected:    "cosmoshub-4",
		},
		{
			name:        "with numbers that are not revision height",
			fullChainID: "cosmoshub-4-1234",
			height:      clientTypes.NewHeight(4, 999999),
			expected:    "cosmoshub-4-1234",
		},
		{
			name:        "without revision number nor revision height",
			fullChainID: "desmos-mainnet",
			height:      clientTypes.NewHeight(4, 999999),
			expected:    "desmos-mainnet",
		},
		{
			name:        "without revision number but with revision height",
			fullChainID: "desmos-mainnet-999999",
			height:      clientTypes.NewHeight(4, 999999),
			expected:    "desmos-mainnet",
		},
		{
			name:        "ignore revision height 0",
			fullChainID: "chain-0",
			height:      clientTypes.NewHeight(4, 0),
			expected:    "chain-0",
		},
		{
			name:        "revision number equals revision height",
			fullChainID: "cosmoshub-4-4",
			height:      clientTypes.NewHeight(4, 4),
			expected:    "cosmoshub-4",
		},
	}

	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			chainID := parseIBCChainID(test.fullChainID, test.height)
			require.Equal(t, test.expected, chainID)
		})
	}
}
