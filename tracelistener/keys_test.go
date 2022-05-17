package tracelistener

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/rand"
)

func TestValidKeys(t *testing.T) {
	t.Parallel()
	randomMaxBytesVal := rand.Bytes(255)
	randomMaxBytesDel := rand.Bytes(255)
	tests := []struct {
		name        string
		key         []byte
		wantDelAddr string
		wantValAddr string
	}{
		{
			name:        "smallest valid address",
			key:         []byte{1, 0, 0},
			wantDelAddr: "",
			wantValAddr: "",
		},
		{
			name:        "largest valid address",
			key:         append(append([]byte{2, 255}, randomMaxBytesDel...), append([]byte{255}, randomMaxBytesVal...)...),
			wantDelAddr: hex.EncodeToString(randomMaxBytesDel),
			wantValAddr: hex.EncodeToString(randomMaxBytesVal),
		},
		{
			name:        "same length address",
			key:         []byte{1, 5, 3, 45, 21, 34, 90, 5, 0, 42, 5, 51, 6},
			wantDelAddr: "032d15225a",
			wantValAddr: "002a053306",
		},
		{
			name:        "variable length addresses",
			key:         []byte{20, 3, 200, 12, 41, 0},
			wantDelAddr: "c80c29",
			wantValAddr: "",
		},
		{
			name:        "hypothetical same address",
			key:         []byte{20, 3, 200, 12, 41, 3, 200, 12, 41},
			wantDelAddr: "c80c29",
			wantValAddr: "c80c29",
		},
		{
			name:        "all zero",
			key:         []byte{20, 3, 0, 0, 0, 3, 0, 0, 0},
			wantDelAddr: "000000",
			wantValAddr: "000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			da, va, err := SplitDelegationKey(tt.key)
			require.NoError(t, err)
			require.Equal(t, da, tt.wantDelAddr)
			require.Equal(t, va, tt.wantValAddr)
		})
	}
}

func TestInValidKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		key    []byte
		errMsg string
	}{
		{
			name:   "empty data slice",
			key:    []byte{},
			errMsg: "malformed key: length 0 not in range",
		},
		{
			name:   "not enough bytes",
			key:    []byte{1, 0},
			errMsg: "malformed key: length 2 not in range",
		},
		{
			name:   "key len out of range",
			key:    append(append([]byte{2, 255}, rand.Bytes(256)...), append([]byte{255}, rand.Bytes(255)...)...),
			errMsg: "malformed key: length 514 not in range",
		},
		{
			name:   "wrong len prefix - less found",
			key:    []byte{1, 5, 3, 45, 21, 34, 90, 6, 0, 42, 5, 51, 6},
			errMsg: "length prefix signals 6 bytes, but total data is 5 bytes long",
		},
		{
			name:   "wrong len prefix - more found",
			key:    []byte{1, 5, 3, 45, 21, 34, 90, 4, 0, 42, 5, 51, 6},
			errMsg: "length prefix signals 4 bytes, but total data is 5 bytes long",
		},
		{
			name:   "wrong len prefix for val address, it has none",
			key:    []byte{1, 5, 3, 45, 21, 34, 90},
			errMsg: "cannot parse validator address, data is nil",
		},
		{
			name:   "delegator address has size but not enough bytes",
			key:    []byte{1, 3, 3},
			errMsg: "delegator address should be 3 bytes long, but it only has 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := SplitDelegationKey(tt.key)
			require.Error(t, err)
			require.ErrorContains(t, err, tt.errMsg)
		})
	}
}

func TestFromLengthPrefix(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		rawData []byte
		want    []byte
		wantErr bool
	}{
		{
			"a length-prefix works",
			[]byte{
				4,          // length prefix
				1, 2, 3, 4, // data
			},
			[]byte{1, 2, 3, 4},
			false,
		},
		{
			"a length-prefix with more data than anticipated",
			[]byte{
				4,             // length prefix
				1, 2, 3, 4, 5, // data
			},
			nil,
			true,
		},
		{
			"a length-prefix with less data than anticipated",
			[]byte{
				4,       // length prefix
				1, 2, 3, // data
			},
			nil,
			true,
		},
		{
			"nil rawData",
			nil,
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			res, err := FromLengthPrefix(tt.rawData)
			if tt.wantErr {
				require.Error(t, err)
				require.Empty(t, res)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, res)
		})
	}
}

func TestSplitCW20BalanceKey(t *testing.T) {
	var (
		// Reference values
		contractAddr             = "ade4a5f5803a439835c636395a8d648dee57b2fc90d98dc17fa887159b69638b"
		holderAddr               = "7761736d313467307339773373797965766b3366347a70327265637470646b376633757371676e35783666"
		holderAddr_bech32Decoded = "aa1f02ba302132cb453510543ce1616dbc98f200"

		// Handy function to build a cw20 balance key
		buildKey = func(prefix []byte, contractAddr string, typ []byte, holderAddr string) []byte {
			ca, _ := hex.DecodeString(contractAddr)
			key := append(prefix, ca...)
			key = append(key, typ...)
			ha, _ := hex.DecodeString(holderAddr)
			key = append(key, ha...)
			return key
		}

		// rawKey ensures the whole test doesn't only rely on the buildKey func
		rawkey, _ = hex.DecodeString("03ade4a5f5803a439835c636395a8d648dee57b2fc90d98dc17fa887159b69638b000762616c616e63657761736d313467307339773373797965766b3366347a70327265637470646b376633757371676e35783666a")
	)
	tests := []struct {
		name                 string
		key                  []byte
		expectedContractAddr string
		expectedHolderAddr   string
		expectedError        string
	}{
		{
			name:          "empty",
			expectedError: "malformed cw20 balance key: length 0 not in range 78-588",
		},
		{
			name:          "too short",
			key:           []byte{1},
			expectedError: "malformed cw20 balance key: length 1 not in range 78-588",
		},
		{
			name:          "too long",
			key:           make([]byte, 1024),
			expectedError: "malformed cw20 balance key: length 1024 not in range 78-588",
		},
		{
			name: "wrong prefix",
			key: buildKey(
				[]byte{42}, contractAddr, wasmContractBalanceKey, holderAddr,
			),
			expectedError: "not a wasm contract store key",
		},
		{
			name: "wrong type",
			key: buildKey(wasmContractStorePrefix, contractAddr,
				append([]byte{0, 7}, []byte("balancx")...),
				holderAddr),
			expectedError: "not a cw20 balance key",
		},
		{
			name: "ok",
			key: buildKey(
				wasmContractStorePrefix, contractAddr,
				wasmContractBalanceKey, holderAddr,
			),
			expectedContractAddr: contractAddr,
			expectedHolderAddr:   holderAddr_bech32Decoded,
		},
		{
			name:                 "ok raw key",
			key:                  rawkey,
			expectedContractAddr: contractAddr,
			expectedHolderAddr:   holderAddr_bech32Decoded,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			contractAddr, holderAddr, err := SplitCW20BalanceKey(tt.key)

			if tt.expectedError != "" {
				assert.EqualError(err, tt.expectedError)
				return
			}
			require.NoError(err)
			assert.Equal(tt.expectedContractAddr, contractAddr)
			assert.Equal(tt.expectedHolderAddr, holderAddr)
		})
	}
}

func TestSplitCW20TokenInfoKey(t *testing.T) {
	var (
		// Reference values
		contractAddr = "ade4a5f5803a439835c636395a8d648dee57b2fc90d98dc17fa887159b69638b"

		// Handy function to build a cw20 token_info key
		buildKey = func(prefix []byte, contractAddr string, typ []byte) []byte {
			ca, _ := hex.DecodeString(contractAddr)
			key := append(prefix, ca...)
			key = append(key, typ...)
			return key
		}

		// rawKey ensures the whole test doesn't only rely on the buildKey func
		rawkey, _ = hex.DecodeString("03ade4a5f5803a439835c636395a8d648dee57b2fc90d98dc17fa887159b69638b746f6b656e5f696e666f")
	)
	tests := []struct {
		name                 string
		key                  []byte
		expectedContractAddr string
		expectedError        string
	}{
		{
			name:          "wrong length",
			expectedError: "malformed cw20 token_info key: length 0 not equal to 43",
		},
		{
			name: "wrong prefix",
			key: buildKey(
				[]byte{42}, contractAddr, wasmContractTokenInfoKey,
			),
			expectedError: "not a wasm contract store key",
		},
		{
			name: "wrong type",
			key: buildKey(
				wasmContractStorePrefix, contractAddr, []byte("token_infx"),
			),
			expectedError: "not a cw20 token_info key",
		},
		{
			name: "ok",
			key: buildKey(
				wasmContractStorePrefix, contractAddr, wasmContractTokenInfoKey,
			),
			expectedContractAddr: contractAddr,
		},
		{
			name:                 "ok raw key",
			key:                  rawkey,
			expectedContractAddr: contractAddr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			contractAddr, err := SplitCW20TokenInfoKey(tt.key)

			if tt.expectedError != "" {
				assert.EqualError(err, tt.expectedError)
				return
			}
			require.NoError(err)
			assert.Equal(tt.expectedContractAddr, contractAddr)
		})
	}
}
