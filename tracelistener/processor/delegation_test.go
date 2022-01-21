package processor

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	models "github.com/allinbits/demeris-backend-models/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener"
	"github.com/allinbits/tracelistener/tracelistener/config"
	"github.com/allinbits/tracelistener/tracelistener/processor/datamarshaler"
)

func TestDelegationOwnsKey(t *testing.T) {
	d := delegationsProcessor{}

	tests := []struct {
		name        string
		prefix      []byte
		key         string
		expectedErr bool
	}{
		{
			"Correct prefix- no error",
			datamarshaler.DelegationKey,
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
				require.False(t, d.OwnsKey(append(tt.prefix, []byte(tt.key)...)))
			} else {
				require.True(t, d.OwnsKey(append(tt.prefix, []byte(tt.key)...)))
			}
		})
	}
}

type testDelegation struct {
	Delegator string
	Validator string
	Shares    int64
}

func TestDelegationProcess(t *testing.T) {
	d := delegationsProcessor{}

	DataProcessor, err := New(zap.NewNop().Sugar(), &config.Config{})
	require.NoError(t, err)

	gp := DataProcessor.(*Processor)
	require.NotNil(t, gp)

	tests := []struct {
		name        string
		delegation  testDelegation
		newMessage  tracelistener.TraceOperation
		expectedErr bool
		expectedLen int
	}{
		{
			"Delete operation of delegation - no error",
			testDelegation{},
			tracelistener.TraceOperation{
				Operation: string(tracelistener.DeleteOp),
				Key:       []byte("QXRkbFY4cUQ2bzZKMnNoc2o5YWNwSSs5T3BkL2U1dVRxWklpN05LNWkzeTk="),
			},
			false,
			1,
		},
		{
			"Write new delegation - no error",
			testDelegation{
				Delegator: "cosmos1xrnner9s783446yz3hhshpr5fpz6wzcwkvwv5j",
				Validator: "cosmosvaloper19xawgvgn887e9gef5vkzkemwh33mtgwa6haa7s",
				Shares:    100,
			},
			tracelistener.TraceOperation{
				Operation:   string(tracelistener.WriteOp),
				Key:         []byte("AtdlV8qD6o6J2shsj9acpI+9Opd/e5uTqZIi7NK5i3y9"),
				BlockHeight: 1,
				TxHash:      "A5CF62609D62ADDE56816681B6191F5F0252D2800FC2C312EB91D962AB7A97CB",
			},
			false,
			1,
		},
		{
			"Invalid addresses - error",
			testDelegation{
				Shares: 100,
			},
			tracelistener.TraceOperation{
				Operation:   string(tracelistener.WriteOp),
				Key:         []byte("AtdlV8qD6o6J2shsj9acpI+9Opd/e5uTqZIi7NK5i3y9"),
				BlockHeight: 1,
				TxHash:      "A5CF62609D62ADDE56816681B6191F5F0252D2800FC2C312EB91D962AB7A97CB",
			},
			true,
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d.insertHeightCache = map[delegationCacheEntry]models.DelegationRow{}
			d.deleteHeightCache = map[delegationCacheEntry]models.DelegationRow{}
			d.l = zap.NewNop().Sugar()

			tt.newMessage.Value = datamarshaler.NewTestDataMarshaler().Delegation(
				tt.delegation.Validator,
				tt.delegation.Delegator,
				tt.delegation.Shares,
			)

			err = d.Process(tt.newMessage)
			if tt.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tt.newMessage.Operation == tracelistener.DeleteOp.String() {
				require.Len(t, d.deleteHeightCache, tt.expectedLen)

				for k := range d.deleteHeightCache {
					row := d.deleteHeightCache[delegationCacheEntry{delegator: k.delegator, validator: k.validator}]
					require.NotNil(t, row)

					return
				}
			} else {
				require.Len(t, d.insertHeightCache, tt.expectedLen)

				for k := range d.insertHeightCache {
					row := d.insertHeightCache[delegationCacheEntry{delegator: k.delegator, validator: k.validator}]
					require.NotNil(t, row)

					amount := row.Amount
					amtfloat, err := strconv.ParseFloat(amount, 64)
					require.NoError(t, err)
					require.EqualValues(t, tt.delegation.Shares, amtfloat)

					return
				}
			}
		})
	}
}

func TestDelegationFlushCache(t *testing.T) {
	d := delegationsProcessor{}

	tests := []struct {
		name        string
		row         models.DelegationRow
		isNil       bool
		expectedNil bool
	}{
		{
			"Non empty data - No error",
			models.DelegationRow{
				Validator: "cosmos1xrnner9s783446yz3hhshpr5fpz6wzcwkvwv5j",
				Delegator: "cosmosvaloper19xawgvgn887e9gef5vkzkemwh33mtgwa6haa7s",
				Amount:    "100stake",
			},
			false,
			false,
		},
		{
			"Empty data - error",
			models.DelegationRow{},
			true,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d.insertHeightCache = map[delegationCacheEntry]models.DelegationRow{}
			d.deleteHeightCache = map[delegationCacheEntry]models.DelegationRow{}

			if !tt.isNil {
				d.insertHeightCache[delegationCacheEntry{
					delegator: tt.row.Delegator,
					validator: tt.row.Validator,
				}] = tt.row

				d.deleteHeightCache[delegationCacheEntry{
					delegator: tt.row.Delegator,
					validator: tt.row.Validator,
				}] = tt.row
			}

			wop := d.FlushCache()
			if tt.expectedNil {
				if len(wop) != 0 {
					require.Empty(t, wop[0].Data)
				}
			} else {
				require.NotNil(t, wop)
			}
		})
	}
}
