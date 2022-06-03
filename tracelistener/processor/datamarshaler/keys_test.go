package datamarshaler

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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

			res, err := fromLengthPrefix(tt.rawData)
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
