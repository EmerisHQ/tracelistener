package tracelistener

import (
	"fmt"
)

const (
	metadataBlockHeight = "blockHeight"
	metadataTxHash      = "txHash"
)

// var toiPool = sync.Pool{
// 	New: func() interface{} {
// 		return &traceOperationInter{}
// 	},
// }

// type TraceOperation struct {
// 	Operation   string `json:"operation"`
// 	Key         []byte `json:"key"`
// 	Value       []byte `json:"value"`
// 	BlockHeight uint64 `json:"block_height"`
// 	TxHash      string `json:"tx_hash"`

// 	// SuggestedProcessor signals to the trace processor that
// 	// what SDK module this trace comes from.
// 	SuggestedProcessor SDKModuleName
// }

func (to *TraceOperation) Copy() TraceOperation {
	ret := TraceOperation{}
	ret = *to
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

func (t TraceOperation) String() string {
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

// func (toi *traceOperationInter) Reset() {
// 	toi.Operation = ""
// 	toi.Key = toi.Key[:0]
// 	toi.Value = toi.Value[:0]
// 	toi.Metadata.BlockHeight = 0
// 	toi.Metadata.TxHash = ""
// }

// func (t *TraceOperation) UnmarshalJSON(bytes []byte) error {
// 	toi := toiPool.Get().(*traceOperationInter)
// 	toi.Reset()

// 	if err := json.Unmarshal(bytes, &toi); err != nil {
// 		return err
// 	}

// 	t.BlockHeight = toi.Metadata.BlockHeight
// 	t.TxHash = toi.Metadata.TxHash
// 	t.Operation = toi.Operation
// 	t.Key = toi.Key
// 	t.Value = toi.Value

// 	toiPool.Put(toi)

// 	return nil
// }
