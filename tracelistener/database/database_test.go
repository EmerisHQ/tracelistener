package database

import (
	"testing"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		connString  string
		expectedErr bool
		startDB     bool
	}{
		{
			name:        "Connecting to database works - no error",
			expectedErr: false,
			startDB:     true,
		},
		{
			name:        "connection string is not valid - error",
			connString:  "invalidconnection",
			expectedErr: true,
			startDB:     false,
		},
		{
			name:        "connection string valid but database is down - error",
			expectedErr: true,
			startDB:     false,
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

			i, err := New(tt.connString)

			if tt.expectedErr {
				require.Error(t, err)
				require.Nil(t, i)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, i)
		})
	}
}

var testDBMigrations = []string{
	`create database testdb`,
	`create table testdb.table (
		id serial primary key,
		first text not null,
		second text not null
	)`,
}
