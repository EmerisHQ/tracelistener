package gaia_processor

import (
	"testing"

	transferTypes "github.com/cosmos/cosmos-sdk/x/ibc/applications/transfer/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	models "github.com/allinbits/demeris-backend-models/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener/config"
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
			transferTypes.DenomTraceKey,
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

func TestIBCDenomTracesProcess(t *testing.T) {
	dtp := ibcDenomTracesProcessor{}

	DataProcessor, err := New(zap.NewNop().Sugar(), &config.Config{})
	require.NoError(t, err)

	gp := DataProcessor.(*Processor)
	require.NotNil(t, gp)
	p.cdc = gp.cdc

	tests := []struct {
		name        string
		newMessage  tracelistener.TraceOperation
		dt          transferTypes.DenomTrace
		expectedEr  bool
		expectedLen int
	}{
		{
			"Add denom trace - no error",
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
			},
			transferTypes.DenomTrace{
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
			transferTypes.DenomTrace{},
			true,
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dtp.denomTracesCache = map[string]models.IBCDenomTraceRow{}
			dtp.l = zap.NewNop().Sugar()

			value, err := p.cdc.MarshalBinaryBare(&tt.dt)
			require.NoError(t, err)
			tt.newMessage.Value = value

			err = dtp.Process(tt.newMessage)
			if tt.expectedEr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// check cache length
			require.Len(t, dtp.denomTracesCache, tt.expectedLen)

			// if denomtrace cache not empty then check the data
			for k, _ := range dtp.denomTracesCache {
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
		path        string
		baseDenom   string
		hash        string
		isNil       bool
		expectedNil bool
	}{
		{
			"Non empty data - No error",
			"path",
			"stake",
			"hash",
			false,
			false,
		},
		{
			"Empty data - error",
			"",
			"",
			"",
			true,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i.denomTracesCache = map[string]models.IBCDenomTraceRow{}

			if !tt.isNil {
				row := models.IBCDenomTraceRow{
					Path:      tt.path,
					BaseDenom: tt.baseDenom,
					Hash:      tt.hash,
				}

				i.denomTracesCache[tt.hash] = row
			}

			wop := i.FlushCache()
			if tt.expectedNil {
				require.Nil(t, wop)
			} else {
				require.NotNil(t, wop)
			}

			return
		})
	}
}
