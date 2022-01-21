package datamarshaler

import (
	models "github.com/allinbits/demeris-backend-models/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener"
	"go.uber.org/zap"
)

// DataMarshaler describes a type which is capable of marshaling/unmarshaling
// Cosmos SDK objects.
type Handler interface {
	Bank(data tracelistener.TraceOperation) (models.BalanceRow, error)
	Auth(data tracelistener.TraceOperation) (models.AuthRow, error)
	Delegations(data tracelistener.TraceOperation) (models.DelegationRow, error)
	IBCChannels(data tracelistener.TraceOperation) (models.IBCChannelRow, error)
	IBCClients(data tracelistener.TraceOperation) (models.IBCClientStateRow, error)
	IBCConnections(data tracelistener.TraceOperation) (models.IBCConnectionRow, error)
	IBCDenomTraces(data tracelistener.TraceOperation) (models.IBCDenomTraceRow, error)
	UnbondingDelegations(data tracelistener.TraceOperation) (models.UnbondingDelegationRow, error)
	Validators(data tracelistener.TraceOperation) (models.ValidatorRow, error)
}

type TestHandler interface {
	AccountBytes(accountNumber, sequenceNumber uint64, address string) []byte
}

// Compile-time check! DataMarshaler must always implement Handler.
// This won't compile if that assumption isn't true.
var _ Handler = DataMarshaler{}
var _ TestHandler = TestDataMarshaler{}

// DataMarshaler is a concrete implementation of Handler.
type DataMarshaler struct {
	l *zap.SugaredLogger
}

func NewDataMarshaler(l *zap.SugaredLogger) DataMarshaler {
	return DataMarshaler{
		l: l,
	}
}

type TestDataMarshaler struct {
	DataMarshaler
}

func NewTestDataMarshaler() TestHandler {
	return TestDataMarshaler{
		DataMarshaler: NewDataMarshaler(zap.NewNop().Sugar()),
	}
}
