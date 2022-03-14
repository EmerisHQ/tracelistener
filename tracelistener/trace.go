package tracelistener

import (
	"fmt"
)

func (to *TraceOperation) Copy() TraceOperation {
	ret := *to
	return ret
}

func (to *TraceOperation) Reset() {
	to.Operation = ""
	to.Key = to.Key[:0]
	to.Value = to.Value[:0]
	to.Metadata.BlockHeight = 0
	to.Metadata.TxHash = ""
	to.SuggestedProcessor = ""
}

func (to TraceOperation) String() string {
	return fmt.Sprintf(`[%s] "%v" -> "%v"`, t.Operation, string(t.Key), string(t.Value))
}

type TraceMetadata struct {
	BlockHeight uint64 `json:"blockHeight"`
	TxHash      string `json:"txHash"`
}

type TraceOperation struct {
	Operation string        `json:"operation"`
	Key       []byte        `json:"key"`
	Value     []byte        `json:"value"`
	Metadata  TraceMetadata `json:"metadata"`

	// SuggestedProcessor signals to the trace processor that
	// what SDK module this trace comes from.
	SuggestedProcessor SDKModuleName
}
