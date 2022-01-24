//go:build sdk_v42

package datamarshaler

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/gogo/protobuf/proto"
)

func marshalIfaceOrPanic(p proto.Message) []byte {
	data, err := getCodec().MarshalInterface(p)
	if err != nil {
		panic(err)
	}

	return data
}

func marshalOrPanic(p codec.ProtoMarshaler) []byte {
	return getCodec().MustMarshalBinaryBare(p)
}
