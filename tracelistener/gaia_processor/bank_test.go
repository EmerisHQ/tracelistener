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

	data := tracelistener.TraceOperation{
		Operation:   string(tracelistener.WriteOp),
		Key:         []byte("YmFsYW5jZXPxgpZ221d2gulE/DST1FG2f/Pin3N0YWtl"),
		Value:       []byte("9588stake"),
		BlockHeight: 101,
	}

	err := b.Process(data)
	require.NoError(t, err)
}
