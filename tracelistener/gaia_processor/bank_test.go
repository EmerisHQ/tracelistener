package gaia_processor

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/allinbits/tracelistener/tracelistener"
)

func TestBankProcess(t *testing.T) {
	b := bankProcessor{}

	require.True(t, b.OwnsKey([]byte("balances500stake")))
	require.False(t, b.OwnsKey([]byte("bal")))

	tests := []struct {
		name       string
		newMessage tracelistener.TraceOperation
		wantErr    bool
	}{
		{
			"No error of bank process",
			tracelistener.TraceOperation{
				Operation:   string(tracelistener.WriteOp),
				Key:         []byte("YmFsYW5jZXPxgpZ221d2gulE/DST1FG2f/Pin3N0YWtl"),
				Value:       []byte("9588stake"),
				BlockHeight: 101,
			},
			false,
		},
		{
			"Invalid value - error",
			tracelistener.TraceOperation{
				Operation:   string(tracelistener.WriteOp),
				Key:         []byte("YmFsYW5jZXPxgpZ221d2gulE/DST1FG2f/Pin3N0YWtl"),
				Value:       []byte("9588"),
				BlockHeight: 101,
			},
			true,
		},
		{
			"Invalid key - error",
			tracelistener.TraceOperation{
				Operation:   string(tracelistener.WriteOp),
				Key:         []byte("pZ221d2gulE/DST1FG2f/Pin3N0YWtl"),
				Value:       []byte("9588"),
				BlockHeight: 101,
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := b.Process(tt.newMessage)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
