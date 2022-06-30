package processor

import (
	"strconv"
	"testing"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/emerishq/tracelistener/models"
	"github.com/emerishq/tracelistener/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener/config"
	"github.com/emerishq/tracelistener/tracelistener/database"
	"github.com/emerishq/tracelistener/tracelistener/processor/datamarshaler"
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
		newMessages []tracelistener.TraceOperation
		expectedErr bool
		expectedLen int
	}{
		{
			"Delete operation of delegation - no error",
			testDelegation{},
			[]tracelistener.TraceOperation{
				{
					Operation: string(tracelistener.DeleteOp),
					// <prefix><9><"delegator"><9><"validator">
					Key: []byte{49, 9, 100, 101, 108, 101, 103, 97, 116, 111, 114, 9, 118, 97, 108, 105, 100, 97, 116, 111, 114},
				},
			},
			false,
			1,
		},
		{
			"Multiple delete operation of delegation - no error",
			testDelegation{},
			[]tracelistener.TraceOperation{
				{
					Operation: string(tracelistener.DeleteOp),
					// <prefix><9><"delegator"><9><"validator">
					Key: []byte{49, 9, 100, 101, 108, 101, 103, 97, 116, 111, 114, 9, 118, 97, 108, 105, 100, 97, 116, 111, 114},
				},
				{
					Operation: string(tracelistener.DeleteOp),
					// <prefix><9><"delegator"><9><"validator">
					Key: []byte{49, 9, 101, 101, 108, 101, 103, 97, 116, 111, 114, 9, 119, 97, 108, 105, 100, 97, 116, 111, 114},
				},
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
			[]tracelistener.TraceOperation{
				{
					Operation:   string(tracelistener.WriteOp),
					Key:         []byte("AtdlV8qD6o6J2shsj9acpI+9Opd/e5uTqZIi7NK5i3y9"),
					BlockHeight: 1,
					TxHash:      "A5CF62609D62ADDE56816681B6191F5F0252D2800FC2C312EB91D962AB7A97CB",
				},
			},
			false,
			1,
		},
		{
			"Invalid addresses - error",
			testDelegation{
				Shares: 100,
			},
			[]tracelistener.TraceOperation{
				{
					Operation:   string(tracelistener.WriteOp),
					Key:         []byte("AtdlV8qD6o6J2shsj9acpI+9Opd/e5uTqZIi7NK5i3y9"),
					BlockHeight: 1,
					TxHash:      "A5CF62609D62ADDE56816681B6191F5F0252D2800FC2C312EB91D962AB7A97CB",
				},
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

			for _, message := range tt.newMessages {
				message.Value = datamarshaler.NewTestDataMarshaler().Delegation(
					tt.delegation.Validator,
					tt.delegation.Delegator,
					tt.delegation.Shares,
				)

				err = d.Process(message)
				if tt.expectedErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}

				if message.Operation == tracelistener.DeleteOp.String() {
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

func Test_Upsert_After_Soft_Delete_Restore_Row(t *testing.T) {
	requireT := require.New(t)
	db, err := prepareDelegationDatabase(t)
	requireT.NoError(err)

	row := models.DelegationRow{
		TracelistenerDatabaseRow: models.TracelistenerDatabaseRow{
			ChainName:    "chain",
			Height:       1,
			DeleteHeight: nil,
		},
		Delegator: "delegator",
		Validator: "validator",
		Amount:    "100stake",
	}

	// Create a delegation
	_, err = db.Instance.DB.NamedExec(
		delegationsTable.Upsert(),
		row,
	)
	requireT.NoError(err)

	// Delete that delegation
	row.Height = 2
	_, err = db.Instance.DB.NamedExec(
		delegationsTable.Delete(),
		row,
	)
	requireT.NoError(err)

	// Create a new delegation with the same validator and delegator
	row.Amount = "42stake"
	row.Height = 3
	_, err = db.Instance.DB.NamedExec(
		delegationsTable.Upsert(),
		row,
	)
	requireT.NoError(err)

	// Assert a single (non soft-deleted) row exists in database
	var result models.DelegationRow
	err = db.Instance.DB.Get(
		&result,
		"SELECT * FROM tracelistener.delegations WHERE delegator_address=$1 AND validator_address=$2 AND delete_height IS NULL",
		"delegator",
		"validator",
	)
	requireT.NoError(err)
	requireT.Equal(uint64(3), result.Height)
	requireT.Equal("42stake", result.Amount)
	requireT.Nil(result.DeleteHeight)
}

func prepareDelegationDatabase(t *testing.T) (*database.Instance, error) {
	ts, err := testserver.NewTestServer()
	if err != nil {
		return nil, err
	}
	t.Cleanup(ts.Stop)

	connString := ts.PGURL().String()
	di, err := database.New(connString)
	if err != nil {
		return nil, err
	}

	// _, err = di.Instance.DB.Exec("CREATE DATABASE tracelistener")
	// if err != nil {
	// 	return nil, err
	// }

	_, err = di.Instance.DB.Exec(delegationsTable.CreateTable())
	if err != nil {
		return nil, err
	}

	return di, nil
}
