package processor

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	models "github.com/emerishq/demeris-backend-models/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener/config"
	"github.com/emerishq/tracelistener/tracelistener/processor/datamarshaler"
)

func TestIbcDenomTracesOwnsKey(t *testing.T) {
	i := ibcDenomTracesProcessor{}

	tests := []struct {
		name        string
		prefix      []byte
		key         string
		expectedErr bool
	}{
		{
			"Correct prefix- no error",
			datamarshaler.IBCDenomTracesKey,
			"key",
			false,
		},
		{
			"Incorrect prefix- error",
			[]byte{0x0},
			"key",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.expectedErr {
				require.False(t, i.OwnsKey(append(tt.prefix, []byte(tt.key)...)))
			} else {
				require.True(t, i.OwnsKey(append(tt.prefix, []byte(tt.key)...)))
			}
		})
	}
}

type testDenomTrace struct {
	Path      string
	BaseDenom string
}

func TestIBCDenomTracesProcess(t *testing.T) {
	dtp := ibcDenomTracesProcessor{}

	DataProcessor, err := New(zap.NewNop().Sugar(), &config.Config{})
	require.NoError(t, err)

	gp := DataProcessor.(*Processor)
	require.NotNil(t, gp)

	tests := []struct {
		name        string
		newMessage  tracelistener.TraceOperation
		dt          testDenomTrace
		expectedEr  bool
		expectedLen int
	}{
		{
			"Add denom trace - no error",
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
			},
			testDenomTrace{
				Path:      "1234/channelId",
				BaseDenom: "stake",
			},
			false,
			1,
		},
		{
			"Base denomination cannot be blank - error",
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
			},
			testDenomTrace{},
			true,
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dtp.denomTracesCache = map[string]models.IBCDenomTraceRow{}
			dtp.l = zap.NewNop().Sugar()

			tt.newMessage.Value = datamarshaler.NewTestDataMarshaler().IBCDenomTraces(
				tt.dt.Path,
				tt.dt.BaseDenom,
			)

			err = dtp.Process(tt.newMessage)
			if tt.expectedEr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// check cache length
			require.Len(t, dtp.denomTracesCache, tt.expectedLen)

			// if denomtrace cache not empty then check the data
			for k := range dtp.denomTracesCache {
				row := dtp.denomTracesCache[k]
				require.NotNil(t, row)

				denom := row.BaseDenom
				require.Equal(t, tt.dt.BaseDenom, denom)

				return
			}
		})
	}
}

func TestIbcDenomTracesFlushCache(t *testing.T) {
	i := ibcDenomTracesProcessor{}

	tests := []struct {
		name        string
		row         models.IBCDenomTraceRow
		isNil       bool
		expectedNil bool
	}{
		{
			"Non empty data - No error",
			models.IBCDenomTraceRow{
				Path:      "path",
				Hash:      "hash",
				BaseDenom: "stake",
			},
			false,
			false,
		},
		{
			"Empty data - error",
			models.IBCDenomTraceRow{},
			true,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i.denomTracesCache = map[string]models.IBCDenomTraceRow{}

			if !tt.isNil {
				i.denomTracesCache[tt.row.Hash] = tt.row
			}

			wop := i.FlushCache()
			if tt.expectedNil {
				require.Nil(t, wop)
			} else {
				require.NotNil(t, wop)
			}
		})
	}
}
