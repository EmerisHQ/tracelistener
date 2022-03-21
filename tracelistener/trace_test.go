package tracelistener_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/emerishq/tracelistener/tracelistener"
)

const (
	noBase64 = `{"operation":"write","key":"key","value":"value"}`

	writeOp           = `{"operation":"write","key":"aGVsbG8K","value":"aGVsbG8K"}`
	readOp            = `{"operation":"read","key":"aGVsbG8K","value":"aGVsbG8K"}`
	deleteOp          = `{"operation":"delete","key":"aGVsbG8K","value":"aGVsbG8K"}`
	iterRangeOp       = `{"operation":"iterRange","key":"aGVsbG8K","value":"aGVsbG8K"}`
	opWithBlockHeight = `{"operation":"write","key":"aGVsbG8K","value":"aGVsbG8K", "metadata": {"blockHeight":42}}`
	opWithTxHash      = `{"operation":"write","key":"aGVsbG8K","value":"aGVsbG8K", "metadata": {"txHash": "hash"}}`
	writeOpNoNewlines = `{"operation":"write","key":"aGVsbG8=","value":"aGVsbG8="}`
)

func TestTraceOperation_UnmarshalJSON(t1 *testing.T) {
	tests := []struct {
		name    string
		data    string
		res     tracelistener.TraceOperation
		wantErr bool
	}{
		{
			"key and value are not base64",
			noBase64,
			tracelistener.TraceOperation{},
			true,
		},
		{
			"garbage data",
			"nope",
			tracelistener.TraceOperation{},
			true,
		},
		{
			"write operation",
			writeOp,
			tracelistener.TraceOperation{
				Operation: "write",
				Key:       []byte{0x68, 0x65, 0x6c, 0x6c, 0x6f, 0xa},
				Value:     []byte{0x68, 0x65, 0x6c, 0x6c, 0x6f, 0xa},
			},
			false,
		},
		{
			"read operation",
			readOp,
			tracelistener.TraceOperation{
				Operation: "read",
				Key:       []byte{0x68, 0x65, 0x6c, 0x6c, 0x6f, 0xa},
				Value:     []byte{0x68, 0x65, 0x6c, 0x6c, 0x6f, 0xa},
			},
			false,
		},
		{
			"delete operation",
			deleteOp,
			tracelistener.TraceOperation{
				Operation: "delete",
				Key:       []byte{0x68, 0x65, 0x6c, 0x6c, 0x6f, 0xa},
				Value:     []byte{0x68, 0x65, 0x6c, 0x6c, 0x6f, 0xa},
			},
			false,
		},
		{
			"iterRange operation",
			iterRangeOp,
			tracelistener.TraceOperation{
				Operation: "iterRange",
				Key:       []byte{0x68, 0x65, 0x6c, 0x6c, 0x6f, 0xa},
				Value:     []byte{0x68, 0x65, 0x6c, 0x6c, 0x6f, 0xa},
			},
			false,
		},
		{
			"operation with block height",
			opWithBlockHeight,
			tracelistener.TraceOperation{
				Operation:   "write",
				Key:         []byte{0x68, 0x65, 0x6c, 0x6c, 0x6f, 0xa},
				Value:       []byte{0x68, 0x65, 0x6c, 0x6c, 0x6f, 0xa},
				BlockHeight: 42,
			},
			false,
		},
		{
			"operation with tx hash",
			opWithTxHash,
			tracelistener.TraceOperation{
				Operation: "write",
				Key:       []byte{0x68, 0x65, 0x6c, 0x6c, 0x6f, 0xa},
				Value:     []byte{0x68, 0x65, 0x6c, 0x6c, 0x6f, 0xa},
				TxHash:    "hash",
			},
			false,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t *testing.T) {
			var to tracelistener.TraceOperation

			err := json.Unmarshal([]byte(tt.data), &to)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.res, to)
		})
	}
}

func TestTraceOperation_String(t1 *testing.T) {
	var to tracelistener.TraceOperation
	require.NoError(t1, json.Unmarshal([]byte(writeOpNoNewlines), &to))

	require.NotEqual(t1, `[write] "[104 101 108 108 111 10]" -> "[104 101 108 108 111 10]"`, to.String())
	require.Equal(t1, `[write] "hello" -> "hello"`, to.String())
}
