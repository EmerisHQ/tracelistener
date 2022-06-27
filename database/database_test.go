package database_test

import (
	"testing"

	"github.com/cockroachdb/cockroach-go/v2/testserver"

	"github.com/stretchr/testify/require"

	"github.com/emerishq/tracelistener/database"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name       string
		connString string
		wantErr    bool
		startDB    bool
	}{
		{
			name:    "database works",
			wantErr: false,
			startDB: true,
		},
		{
			name:       "connection string is not valid",
			connString: "invalid",
			wantErr:    true,
			startDB:    false,
		},
		{
			name:    "connection string valid but database is down",
			wantErr: true,
			startDB: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.startDB {
				ts, err := testserver.NewTestServer()
				require.NoError(t, err)
				require.NoError(t, ts.WaitForInit())
				defer func() {
					ts.Stop()
				}()

				if tt.connString == "" {
					tt.connString = ts.PGURL().String()
				}
			}

			i, err := database.New(tt.connString)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, i)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, i)
		})
	}
}

func TestClose(t *testing.T) {
	ts, err := testserver.NewTestServer()
	require.NoError(t, err)
	require.NoError(t, ts.WaitForInit())
	defer func() {
		ts.Stop()
	}()

	i, err := database.New(ts.PGURL().String())
	require.NoError(t, err)
	require.NotNil(t, i)

	require.NoError(t, i.Close())
}

var testDBMigrations = []string{
	`create database testdb`,
	`create table testdb.table (
		id serial primary key,
		first text not null,
		second text not null
	)`,
}

func TestRunMigrations(t *testing.T) {
	ts, err := testserver.NewTestServer()
	require.NoError(t, err)
	require.NoError(t, ts.WaitForInit())
	defer func() {
		ts.Stop()
	}()

	require.NoError(t, database.RunMigrations(ts.PGURL().String(), testDBMigrations))
}

func TestExec(t *testing.T) {
	type data struct {
		query  string
		params interface{}
		dest   interface{}
	}

	type fs struct {
		ID     uint64 `db:"id"`
		First  string `db:"first"`
		Second string `db:"second"`
	}

	tests := []struct {
		name    string
		wantErr bool
		data    []data
	}{
		{
			name:    "insert some data",
			wantErr: false,
			data: []data{
				{
					query: "insert into testdb.table (first, second) values (:first, :second)",
					params: map[string]interface{}{
						"first":  "first",
						"second": "second",
					},
				},
			},
		},
		{
			name:    "insert some data, query it back",
			wantErr: false,
			data: []data{
				{
					query: "insert into testdb.table (first, second) values (:first, :second)",
					params: map[string]interface{}{
						"first":  "first",
						"second": "second",
					},
				},
				{
					query: "select * from testdb.table",
					dest:  &[]fs{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, err := testserver.NewTestServer()
			require.NoError(t, err)
			require.NoError(t, ts.WaitForInit())
			defer func() {
				ts.Stop()
			}()

			i, err := database.New(ts.PGURL().String())

			require.NoError(t, err)
			require.NotNil(t, i)

			require.NoError(t, database.RunMigrations(ts.PGURL().String(), testDBMigrations))

			for ii, d := range tt.data {
				hasDest := d.dest != nil
				require.NoError(t, i.Exec(d.query, d.params, d.dest), "iteration %d", ii)

				if hasDest {
					require.NotNil(t, d.dest, "iteration %d", ii)
				}
			}
		})
	}
}
