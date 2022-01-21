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
	Account(accountNumber, sequenceNumber uint64, address string) []byte
	Coin(denom string, amount int64) []byte
	Delegation(validator, delegator string, shares int64) []byte
	IBCChannel(state, ordering int32, counterPortID, counterChannelID string, hop string) []byte
	IBCClient(state TestClientState) []byte
	IBCConnection(conn TestConnection) []byte
	MapConnectionState(s int32) string
	IBCDenomTraces(path, baseDenom string) []byte
	Validator(v TestValidator) []byte
	UnbondingDelegation(u TestUnbondingDelegation) []byte
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
