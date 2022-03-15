package tracelistener

// Copy deep-copies to to a new instances of TraceOperation,
// useful when sending over data down the processing pipeline.
func (to *TraceOperation) Copy() TraceOperation {
	ret := *to

	// Explicitly copy key and value slices to new
	// slice instances to avoid aliasing.
	ret.Key = make([]byte, len(to.Key))
	copy(ret.Key, to.Key)

	ret.Value = make([]byte, len(to.Value))
	copy(ret.Value, to.Value)

	return ret
}

// Reset resets to to an empty state.
// Useful when storing it in a sync.Pool.
func (to *TraceOperation) Reset() {
	to.Operation = ""
	to.Key = to.Key[:0]
	to.Value = to.Value[:0]
	to.Metadata.BlockHeight = 0
	to.Metadata.TxHash = ""
	to.SuggestedProcessor = ""
}

// TraceMetadata holds circumstantial information about a trace,
// like the block height at which it was generated, and optionally a
// the block hash that generated it.
type TraceMetadata struct {
	BlockHeight uint64 `json:"blockHeight"`
	TxHash      string `json:"txHash"`
}

// TraceOperation represents a Cosmos SDK store operation, parsed from
// JSON lines produced by the SDK's "--trace-store" CLI flag.
type TraceOperation struct {
	Operation string        `json:"operation"`
	Key       []byte        `json:"key"`
	Value     []byte        `json:"value"`
	Metadata  TraceMetadata `json:"metadata"`

	// SuggestedProcessor signals to the trace processor that
	// what SDK module this trace comes from.
	SuggestedProcessor SDKModuleName
}
