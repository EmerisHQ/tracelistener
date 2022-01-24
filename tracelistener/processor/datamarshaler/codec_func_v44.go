//go:build sdk_v44

package datamarshaler

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/gogo/protobuf/proto"
)

// This file contains implementation for Cosmos SDK v0.42.x for marshaling interfaces.
// Those functions are often used in impl_test_handler.go.

func marshalIfaceOrPanic(p proto.Message) []byte {
	data, err := getCodec().MarshalInterface(p)
	if err != nil {
		panic(err)
	}

	return data
}

func marshalOrPanic(p codec.ProtoMarshaler) []byte {
	return getCodec().MustMarshal(p)
}
