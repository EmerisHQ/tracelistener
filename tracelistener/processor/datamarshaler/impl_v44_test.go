//go:build sdk_v44

package datamarshaler

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank/types"
	clientTypes "github.com/cosmos/ibc-go/v2/modules/core/02-client/types"
	models "github.com/emerishq/demeris-backend-models/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestDataMarshalerBank(t *testing.T) {
	var (
		coins = sdk.Coin{
			Denom:  "uatom",
			Amount: sdk.NewInt(100),
		}
		coinBz, _ = coins.Marshal()
	)
	tests := []struct {
		name               string
		tr                 tracelistener.TraceOperation
		expectedError      string
		expectedBalanceRow models.BalanceRow
	}{
		{
			name: "fail: key doesn't have address",
			tr: tracelistener.TraceOperation{
				Operation: tracelistener.WriteOp.String(),
				Key:       types.BalancesPrefix,
			},
			expectedError: "cannot parse address from balance store key, invalid key",
		},
		{
			name: "fail: key have wrong address length",
			tr: tracelistener.TraceOperation{
				Operation: tracelistener.WriteOp.String(),
				Key:       append(types.BalancesPrefix, []byte{4, 'a', 'd', 'd'}...),
			},
			expectedError: "cannot parse address from balance store key, invalid key",
		},
		{
			name: "fail: value is empty",
			tr: tracelistener.TraceOperation{
				Operation: tracelistener.WriteOp.String(),
				Key:       append(types.BalancesPrefix, []byte{3, 'a', 'd', 'd'}...),
			},
			expectedError: "invalid balance coin: invalid denom: ",
		},
		{
			name: "ok: value is not a valid coin",
			tr: tracelistener.TraceOperation{
				Operation: tracelistener.WriteOp.String(),
				Key:       append(types.BalancesPrefix, []byte{3, 'a', 'd', 'd'}...),
				Value:     []byte("\n$\n\x05uiris\x12\x1b509625143506063788050678455"),
			},
			expectedError: "invalid balance coin: invalid denom: \n\x05uiris\x12\x1b509625143506063788050678455",
		},
		{
			name: "ok: value is not a valid coin but operation is delete",
			tr: tracelistener.TraceOperation{
				Operation: tracelistener.DeleteOp.String(),
				Key:       append(types.BalancesPrefix, []byte{3, 'a', 'd', 'd'}...),
			},
			expectedBalanceRow: models.BalanceRow{
				Address: "616464",
				Amount:  "0",
			},
		},
		{
			name: "ok",
			tr: tracelistener.TraceOperation{
				Operation: tracelistener.WriteOp.String(),
				Key:       append(types.BalancesPrefix, []byte{3, 'a', 'd', 'd'}...),
				Value:     coinBz,
			},
			expectedBalanceRow: models.BalanceRow{
				Address: "616464",
				Amount:  "100uatom",
				Denom:   "uatom",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)
			dm := NewDataMarshaler(zap.L().Sugar())

			row, err := dm.Bank(tt.tr)

			if tt.expectedError != "" {
				require.EqualError(err, tt.expectedError)
				return
			}
			require.NoError(err)
			assert.Equal(tt.expectedBalanceRow, row)
		})
	}

}

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
