//go:build sdk_v44

package processor

import (
	"encoding/hex"
	"testing"

	models "github.com/emerishq/demeris-backend-models/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCW20BalanceProcessor(t *testing.T) {
	var (
		balanceKey, _ = hex.DecodeString("03ade4a5f5803a439835c636395a8d648dee57b2fc90d98dc17fa887159b69638b000762616c616e63657761736d313467307339773373797965766b3366347a70327265637470646b376633757371676e35783666a")
		contractAddr  = "ade4a5f5803a439835c636395a8d648dee57b2fc90d98dc17fa887159b69638b"
		holderAddr    = "aa1f02ba302132cb453510543ce1616dbc98f200"
	)
	tests := []struct {
		name                string
		data                tracelistener.TraceOperation
		expectedHeightCache map[cw20BalanceCacheEntry]models.CW20BalanceRow
	}{
		{
			name: "ok",
			data: tracelistener.TraceOperation{
				Key:         balanceKey,
				Value:       []byte("1000"),
				BlockHeight: 42,
			},
			expectedHeightCache: map[cw20BalanceCacheEntry]models.CW20BalanceRow{
				{
					contractAddress: contractAddr,
					address:         holderAddr,
				}: {
					ContractAddress: contractAddr,
					Address:         holderAddr,
					Amount:          "1000",
					TracelistenerDatabaseRow: models.TracelistenerDatabaseRow{
						Height: 42,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)
			p := cw20BalanceProcessor{
				heightCache: map[cw20BalanceCacheEntry]models.CW20BalanceRow{},
			}

			// test OwnsKey
			owned := p.OwnsKey(tt.data.Key)

			require.True(owned, "processor doesn't own key")

			// test Process
			err := p.Process(tt.data)

			require.NoError(err)
			assert.Equal(tt.expectedHeightCache, p.heightCache)
		})
	}
}
