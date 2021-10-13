package moduleprocessor

import (
	"bytes"
	"context"

	sdkserviceclient "github.com/allinbits/sdk-service-meta/gen/grpc/sdk_utilities/client"
	sdkutilities "github.com/allinbits/sdk-service-meta/gen/sdk_utilities"

	tracelistener2 "github.com/allinbits/tracelistener"
	models "github.com/allinbits/demeris-backend-models/tracelistener"
	"google.golang.org/grpc"

	transferTypes "github.com/cosmos/cosmos-sdk/x/ibc/applications/transfer/types"
	"go.uber.org/zap"
)

type ibcDenomTracesProcessor struct {
	l                *zap.SugaredLogger
	grpcConn         *grpc.ClientConn
	denomTracesCache map[string]models.IBCDenomTraceRow
}

func (*ibcDenomTracesProcessor) TableSchema() string {
	return createDenomTracesTable
}

func (b *ibcDenomTracesProcessor) ModuleName() string {
	return "ibc_denom_traces"
}

func (b *ibcDenomTracesProcessor) FlushCache() []tracelistener2.WritebackOp {
	if len(b.denomTracesCache) == 0 {
		return nil
	}

	l := make([]models.DatabaseEntrier, 0, len(b.denomTracesCache))

	for _, c := range b.denomTracesCache {
		l = append(l, c)
	}

	b.denomTracesCache = map[string]models.IBCDenomTraceRow{}

	return []tracelistener2.WritebackOp{
		{
			DatabaseExec: insertDenomTrace,
			Data:         l,
		},
	}
}

func (b *ibcDenomTracesProcessor) OwnsKey(key []byte) bool {
	return bytes.HasPrefix(key, transferTypes.DenomTraceKey)
}

func (b *ibcDenomTracesProcessor) Process(data tracelistener2.TraceOperation) error {
	client := sdkserviceclient.NewClient(b.grpcConn)

	cc := sdkutilities.Client{
		IbcDenomTraceEndpoint: client.IbcDenomTrace(),
	}

	payload := sdkutilities.TracePayload{
		Key:           data.Key,
		Value:         data.Value,
		OperationType: &data.Operation,
	}

	res, err := cc.IbcDenomTrace(context.Background(), &sdkutilities.IbcDenomTracePayload{
		Payload: []*sdkutilities.TracePayload{
			&payload,
		},
	})

	for _, r := range res {
		b.denomTracesCache[r.Hash] = models.IBCDenomTraceRow{
			Path:      r.Path,
			BaseDenom: r.BaseDenom,
			Hash:      r.Hash,
		}
	}

	if err != nil {
		return unwindErrors(err)
	}

	return nil
}
