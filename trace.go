package tracelistener

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

const (
	metadataBlockHeight = "blockHeight"
	metadataTxHash      = "txHash"
)

type TraceOperation struct {
	Operation   string `json:"operation"`
	Key         []byte `json:"key"`
	Value       []byte `json:"value"`
	BlockHeight uint64 `json:"block_height"`
	TxHash      string `json:"tx_hash"`
}

func (t TraceOperation) String() string {
	return fmt.Sprintf(`[%s] "%v" -> "%v"`, t.Operation, t.Key, t.Value)
}

type traceOperationInter struct {
	Operation string                 `json:"operation"`
	Key       string                 `json:"key"`
	Value     string                 `json:"value"`
	Metadata  map[string]interface{} `json:"metadata"`
}

func (t *TraceOperation) UnmarshalJSON(bytes []byte) error {
	toi := traceOperationInter{}

	if err := json.Unmarshal(bytes, &toi); err != nil {
		return err
	}

	if toi.Metadata == nil {
		t.BlockHeight = 0
	} else {
		if data, ok := toi.Metadata[metadataBlockHeight]; ok {
			t.BlockHeight = uint64(data.(float64))
		}

		if data, ok := toi.Metadata[metadataTxHash]; ok {
			t.TxHash = data.(string)
		}
	}

	t.Operation = toi.Operation
	var err error
	t.Key, err = base64.StdEncoding.DecodeString(toi.Key)
	if err != nil {
		return err
	}

	t.Value, err = base64.StdEncoding.DecodeString(toi.Value)
	if err != nil {
		return err
	}

	return nil
}
