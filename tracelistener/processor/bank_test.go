package processor

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	gaia "github.com/cosmos/gaia/v6/app"
	models "github.com/emerishq/demeris-backend-models/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener/config"
	"github.com/emerishq/tracelistener/tracelistener/processor/datamarshaler"
)

func TestFix794(t *testing.T) {
	ops := []string{
		// Iris1
		"{\"operation\":\"write\",\"key\":\"AhT+5cRMA/vuTTyGA9Qqcf/L5yTzpA==\",\"value\":\"CiQKBXVpcmlzEhs1MDk2MjUxNDM1MDYwNjM3ODgwNTA2Nzg0NTU=\",\"metadata\":{\"blockHeight\":15065683,\"txHash\":\"31647EB774FDC743E067FA459AA05CD5B0F315431CCCA54F98D877D7C26BCFC4\"}}",
		// Iris2
		"{\"operation\":\"write\",\"key\":\"AhT+5cRMA/vuTTyGA9Qqcf/L5yTzpA==\",\"value\":\"CiQKBXVpcmlzEhs1MDk2MjU4MDE3MjExOTM0Njg5NTk2NDM3Njk=\",\"metadata\":{\"blockHeight\":15065683,\"txHash\":\"31647EB774FDC743E067FA459AA05CD5B0F315431CCCA54F98D877D7C26BCFC4\"}}",
		// Iris3 read
		"{\"operation\":\"read\",\"key\":\"AhT+5cRMA/vuTTyGA9Qqcf/L5yTzpA==\",\"value\":\"CiQKBXVpcmlzEhs1MDk2MjUxNDM1MDYwNjM3ODgwNTA2Nzg0NTU=\",\"metadata\":{\"blockHeight\":15065683,\"txHash\":\"31647EB774FDC743E067FA459AA05CD5B0F315431CCCA54F98D877D7C26BCFC4\"}}",
		// osmosis1
		"{\"operation\":\"write\",\"key\":\"AhTxgpZ221d2gulE/DST1FG2f/Pin3Vvc21v\",\"value\":\"CgV1b3NtbxIFMTExODI=\",\"metadata\":{\"blockHeight\":4533309,\"txHash\":\"989E5A9E0B87B7A7ED696E965A0CCD66B97E493F30EA102101DEE2807B4C875A\"}}",
		//osmosis2
		"{\"operation\":\"write\",\"key\":\"AhSgqrCihDYKl/VG5eiryfgJBa0QtHVvc21v\",\"value\":\"CgV1b3NtbxIHNzI2NTM5MA==\",\"metadata\":{\"blockHeight\":4533309,\"txHash\":\"989E5A9E0B87B7A7ED696E965A0CCD66B97E493F30EA102101DEE2807B4C875A\"}}",
		// gaia1
		"{\"operation\":\"write\",\"key\":\"AhT7U2l8ff5EAeaLXC+PAenSuJDN1HVhdG9t\",\"value\":\"CgV1YXRvbRIFNTY0Mzc=\",\"metadata\":{\"blockHeight\":10625077,\"txHash\":\"03CCF5BEA0D76759CD7DB8674BC243E00DBB050A56C7A7D983F270BFF17F1DC0\"}}",
		// crescent1
		"{\"operation\":\"write\",\"key\":\"AhSTNUhFAwJ0zUvxaGq9YKso7FLhp3VjcmU=\",\"value\":\"CgR1Y3JlEgw0MTQyNjkzMzA4NTQ=\",\"metadata\":{\"blockHeight\":607479,\"txHash\":\"F02890B45998C1E70628D91421021E51B81375308B95793E8F1D2DED26DCE508\"}}",
	}
	d := bankProcessor{}
	for i := 0; i < len(ops); i++ {
		var tr tracelistener.TraceOperation
		err := json.Unmarshal([]byte(ops[i]), &tr)
		require.NoError(t, err)
		// fmt.Printf("TR %#v\n", tr)
		// fmt.Println("OWN", d.OwnsKey(tr.Key))
		if !d.OwnsKey(tr.Key) {
			continue
		}

		addrBytes, err := bankTypes.AddressFromBalancesStore(tr.Key[1:])
		require.NoError(t, err)

		hAddr := hex.EncodeToString(addrBytes)

		coin := sdk.Coin{
			Amount: sdk.NewInt(0),
		}
		err = gaia.MakeEncodingConfig().Marshaler.Unmarshal(tr.Value, &coin)
		require.NoError(t, err)
		fmt.Printf("%d) KEY=%q COINS %q %q\n", i, hAddr, coin.Denom, coin.Amount)
	}

}

func TestBankProcessorOwnsKey(t *testing.T) {
	d := bankProcessor{}
	tests := []struct {
		name        string
		prefix      []byte
		key         string
		expectedErr bool
	}{
		{
			"Correct prefix- no error",
			datamarshaler.BankKey,
			"key",
			false,
		},
		{
			"Incorrect prefix- error",
			[]byte{0x0},
			"key",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.expectedErr {
				require.False(t, d.OwnsKey(append(tt.prefix, []byte(tt.key)...)))
			} else {
				require.True(t, d.OwnsKey(append(tt.prefix, []byte(tt.key)...)))
			}
		})
	}
}

type testCoin struct {
	Denom  string
	Amount int64
}

func TestBankProcess(t *testing.T) {
	b := bankProcessor{}

	DataProcessor, err := New(zap.NewNop().Sugar(), &config.Config{})
	require.NoError(t, err)

	gp := DataProcessor.(*Processor)
	require.NotNil(t, gp)

	tests := []struct {
		name        string
		coin        testCoin
		newMessage  tracelistener.TraceOperation
		expectedErr bool
		expectedLen int
	}{
		{
			"No error of bank process",
			testCoin{
				Denom:  "stake",
				Amount: 500,
			},
			tracelistener.TraceOperation{
				Operation:   string(tracelistener.WriteOp),
				Key:         []byte("cosmos1xrnner9s783446yz3hhshpr5fpz6wzcwkvwv5j"),
				BlockHeight: 101,
			},
			false,
			1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b.heightCache = map[bankCacheEntry]models.BalanceRow{}
			b.l = zap.NewNop().Sugar()

			tt.newMessage.Key = datamarshaler.NewTestDataMarshaler().BankAddress(string(tt.newMessage.Key))
			tt.newMessage.Value = datamarshaler.NewTestDataMarshaler().Coin(
				tt.coin.Denom,
				tt.coin.Amount,
			)

			err = b.Process(tt.newMessage)
			if tt.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Len(t, b.heightCache, tt.expectedLen)

			for k := range b.heightCache {
				row := b.heightCache[bankCacheEntry{address: k.address, denom: k.denom}]
				require.NotNil(t, row)

				denom := row.Denom
				require.Equal(t, tt.coin.Denom, denom)

				require.Equal(t, tt.newMessage.BlockHeight, row.Height)
			}
		})
	}
}

func TestBankFlushCache(t *testing.T) {
	b := bankProcessor{}

	tests := []struct {
		name        string
		row         models.BalanceRow
		isNil       bool
		expectedNil bool
	}{
		{
			"Non empty data - No error",
			models.BalanceRow{
				Address: "0A1E9FBE949F06AA6CABABF9262EF5C071DCA7E2",
				Denom:   "stake",
				Amount:  "100stake",
			},
			false,
			false,
		},
		{
			"Empty data - error",
			models.BalanceRow{},
			true,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b.heightCache = map[bankCacheEntry]models.BalanceRow{}

			if !tt.isNil {
				b.heightCache[bankCacheEntry{
					address: tt.row.Address,
					denom:   tt.row.Denom,
				}] = tt.row
			}

			wop := b.FlushCache()
			if tt.expectedNil {
				require.Nil(t, wop)
			} else {
				require.NotNil(t, wop)
			}
		})
	}
}
