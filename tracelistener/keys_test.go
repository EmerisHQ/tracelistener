package tracelistener

import (
	"encoding/hex"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/rand"
	"testing"
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
			_, da, va, err := SplitDelegationKey(tt.key)
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
			errMsg: "malformed key: validator. want: 6 got: 5",
		},
		{
			name:   "wrong len prefix - more found",
			key:    []byte{1, 5, 3, 45, 21, 34, 90, 4, 0, 42, 5, 51, 6},
			errMsg: "malformed key: validator. want: 4 got: 5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, _, err := SplitDelegationKey(tt.key)
			require.Error(t, err)
			require.ErrorContains(t, err, tt.errMsg)
		})
	}
}