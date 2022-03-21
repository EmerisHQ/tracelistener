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

type unbondingDelegationsOwnsKeyTest struct {
	name        string
	prefix      []byte
	key         string
	expectedErr bool
}

func TestUnbondingDelegationOwnsKey(t *testing.T) {
	u := unbondingDelegationsProcessor{}

	tests := []unbondingDelegationsOwnsKeyTest{
		{
			"Correct prefix- no error",
			datamarshaler.UnbondingDelegationKey,
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

	tests = append(tests, versionSpecificUnbondingDelegationsOwnsKeyTests()...)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.expectedErr {
				require.False(t, u.OwnsKey(append(tt.prefix, []byte(tt.key)...)))
			} else {
				require.True(t, u.OwnsKey(append(tt.prefix, []byte(tt.key)...)))
			}
		})
	}
}

type unbondingDelegationsProcessTest struct {
	name                string
	unbondingDelegation datamarshaler.TestUnbondingDelegation
	newMessage          tracelistener.TraceOperation
	expectedEr          bool
	expectedLen         int
}

func TestUnbondingDelegationProcess(t *testing.T) {
	u := unbondingDelegationsProcessor{}

	DataProcessor, err := New(zap.NewNop().Sugar(), &config.Config{})
	require.NoError(t, err)

	gp := DataProcessor.(*Processor)
	require.NotNil(t, gp)

	tests := []unbondingDelegationsProcessTest{
		{
			"Write unbonding delegation - no error",
			datamarshaler.TestUnbondingDelegation{
				Delegator: "cosmos1xrnner9s783446yz3hhshpr5fpz6wzcwkvwv5j",
				Validator: "cosmosvaloper19xawgvgn887e9gef5vkzkemwh33mtgwa6haa7s",
				Entries: []datamarshaler.TestUnbondingDelegationEntry{
					{
						Height:         4120,
						InitialBalance: 1000,
						Balance:        1100,
					},
				},
			},
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
				Key:       []byte("AtdlV8qD6o6J2shsj9acpI+9Opd/e5uTqZIi7NK5i3y9"),
				Metadata: tracelistener.TraceMetadata{
					BlockHeight: 1,
					TxHash:      "066050E449C3450F943FC6227F155C19EF5C14653F268E9BAFEFE93DF9B3EDAD",
				},
			},
			false,
			1,
		},
		{
			"Invalid addresses - error",
			datamarshaler.TestUnbondingDelegation{},
			tracelistener.TraceOperation{
				Operation: string(tracelistener.WriteOp),
				Key:       []byte("AtdlV8qD6o6J2shsj9acpI+9Opd/e5uTqZIi7NK5i3y9"),
				Metadata: tracelistener.TraceMetadata{
					BlockHeight: 1,
					TxHash:      "A5CF62609D62ADDE56816681B6191F5F0252D2800FC2C312EB91D962AB7A97CB",
				},
			},
			true,
			0,
		},
	}

	tests = append(tests, versionSpecificUnbondingDelegationsProcessTests()...)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u.insertHeightCache = map[unbondingDelegationCacheEntry]models.UnbondingDelegationRow{}
			u.deleteHeightCache = map[unbondingDelegationCacheEntry]models.UnbondingDelegationRow{}
			u.l = zap.NewNop().Sugar()

			tt.newMessage.Value = datamarshaler.NewTestDataMarshaler().UnbondingDelegation(tt.unbondingDelegation)

			err = u.Process(tt.newMessage)
			if tt.expectedEr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tt.newMessage.Operation == tracelistener.DeleteOp.String() {
				require.Len(t, u.deleteHeightCache, tt.expectedLen)

				for k := range u.deleteHeightCache {
					row := u.deleteHeightCache[unbondingDelegationCacheEntry{delegator: k.delegator, validator: k.validator}]
					require.NotNil(t, row)

					return
				}
			} else {
				require.Len(t, u.insertHeightCache, tt.expectedLen)

				for k := range u.insertHeightCache {
					row := u.insertHeightCache[unbondingDelegationCacheEntry{delegator: k.delegator, validator: k.validator}]
					require.NotNil(t, row)

					delegator := row.Delegator
					delegatorAddr, err := b32Hex(tt.unbondingDelegation.Delegator)
					require.NoError(t, err)

					require.Equal(t, delegatorAddr, delegator)

					return
				}
			}
		})
	}
}

func TestUnbondingDelegationFlushCache(t *testing.T) {
	ud := unbondingDelegationsProcessor{}

	tests := []struct {
		name        string
		row         models.UnbondingDelegationRow
		isNil       bool
		expectedNil bool
	}{
		{
			"Non empty data - No error",
			models.UnbondingDelegationRow{
				Delegator: "cosmos1xrnner9s783446yz3hhshpr5fpz6wzcwkvwv5j",
				Validator: "cosmosvaloper19xawgvgn887e9gef5vkzkemwh33mtgwa6haa7s",
			},
			false,
			false,
		},
		{
			"Empty data - error",
			models.UnbondingDelegationRow{},
			false,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ud.insertHeightCache = map[unbondingDelegationCacheEntry]models.UnbondingDelegationRow{}
			ud.deleteHeightCache = map[unbondingDelegationCacheEntry]models.UnbondingDelegationRow{}

			if !tt.isNil {
				ud.insertHeightCache[unbondingDelegationCacheEntry{
					delegator: tt.row.Delegator,
					validator: tt.row.Validator,
				}] = tt.row

				ud.deleteHeightCache[unbondingDelegationCacheEntry{
					delegator: tt.row.Delegator,
					validator: tt.row.Validator,
				}] = tt.row
			}

			wop := ud.FlushCache()
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
