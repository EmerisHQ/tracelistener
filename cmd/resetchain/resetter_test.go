package main

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/allinbits/emeris-utils/database"
	"github.com/cockroachdb/cockroach-go/v2/testserver"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

var l *zap.SugaredLogger = zap.NewExample().Sugar()

func getTestInstance(t *testing.T) *database.Instance {
	assert := assert.New(t)

	ts, err := testserver.NewTestServer()
	assert.NoError(err)
	t.Cleanup(ts.Stop)

	url := ts.PGURL()
	assert.NotEmpty(url)

	db, err := database.New(url.String())
	assert.NoError(err)
	t.Cleanup(func() {
		err := db.Close()
		assert.NoError(err)
	})

	return db
}

func createTables(t *testing.T, db *database.Instance, tableNames ...string) {
	for _, name := range tableNames {
		_, err := db.DB.Exec(fmt.Sprintf(`
			CREATE TABLE %s (id serial, chain_name text)
		`, name))
		assert.NoError(t, err)
	}
}

func addRows(t *testing.T, db *database.Instance, tableName, chainName string, rowCount int) {
	for i := 0; i < rowCount; i++ {
		_, err := db.DB.Exec(fmt.Sprintf("INSERT INTO %s VALUES ($1, $2)", tableName), i, chainName)
		assert.NoError(t, err)
	}
}

func countRows(t *testing.T, db *database.Instance, tableName, chainName string) int {
	var count int
	err := db.DB.Get(&count, fmt.Sprintf("SELECT count(*) from %s WHERE chain_name = $1", tableName), chainName)
	assert.NoError(t, err)
	return count
}

func TestResetTable_ChunkSize(t *testing.T) {
	db := getTestInstance(t)

	tests := []struct {
		name      string
		chunkSize int
	}{
		{
			name:      "small chunk size (1)",
			chunkSize: 1,
		},
		{
			name:      "large chunk size (10k)",
			chunkSize: 10000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			// arrange
			// create a random new table for each test so we know we start clean
			tableName := fmt.Sprintf("test_table_%d", rand.Int())
			t.Log("created table", tableName)

			createTables(t, db, tableName)
			addRows(t, db, tableName, "chain-a", 13)
			addRows(t, db, tableName, "chain-b", 7)

			count := countRows(t, db, tableName, "chain-a")
			assert.Equal(13, count)

			// act
			err := ResetTable(l, db.DB, tableName, "chain-a", 1)
			assert.NoError(err)

			// assert
			count = countRows(t, db, tableName, "chain-a")
			assert.Equal(0, count, "chain-a row not deleted")
			countB := countRows(t, db, tableName, "chain-b")
			assert.Equal(7, countB, "chain-b rows deleted but they were not supposed to")
		})
	}
}

func TestResetTable_IgnoreNonExistentTables(t *testing.T) {
	db := getTestInstance(t)
	err := ResetTable(l, db.DB, "something_non_existent", "chain-a", 1)
	assert.NoError(t, err)
}
