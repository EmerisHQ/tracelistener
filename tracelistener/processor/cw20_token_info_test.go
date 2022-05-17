package processor

import (
	"encoding/hex"
	"testing"

	models "github.com/emerishq/demeris-backend-models/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCW20TokenInfoProcessor(t *testing.T) {
	var (
		tokenInfoKey, _ = hex.DecodeString("03ade4a5f5803a439835c636395a8d648dee57b2fc90d98dc17fa887159b69638b746f6b656e5f696e666f")
		contractAddr    = "ade4a5f5803a439835c636395a8d648dee57b2fc90d98dc17fa887159b69638b"
		tokenInfoValue  = []byte(`{
    "name": "meme",
    "symbol": "umeme",
    "decimals": 18,
    "total_supply": "169420"
}`)
	)
	tests := []struct {
		name                string
		data                tracelistener.TraceOperation
		expectedHeightCache map[cw20TokenInfoCacheEntry]models.CW20TokenInfoRow
	}{
		{
			name: "ok",
			data: tracelistener.TraceOperation{
				Key:         tokenInfoKey,
				Value:       tokenInfoValue,
				BlockHeight: 42,
			},
			expectedHeightCache: map[cw20TokenInfoCacheEntry]models.CW20TokenInfoRow{
				{
					contractAddress: contractAddr,
				}: {
					ContractAddress: contractAddr,
					Name:            "meme",
					Symbol:          "umeme",
					Decimals:        18,
					TotalSupply:     "169420",
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
			p := cw20TokenInfoProcessor{
				heightCache: map[cw20TokenInfoCacheEntry]models.CW20TokenInfoRow{},
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
