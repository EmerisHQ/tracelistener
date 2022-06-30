package database

import dbutils "github.com/emerishq/tracelistener/database"

const createDatabase = `
CREATE DATABASE IF NOT EXISTS tracelistener;
`

var migrationList = []string{
	createDatabase,
}

func (i *Instance) runMigrations() {
	if err := dbutils.RunMigrations(i.connString, migrationList); err != nil {
		panic(err)
	}
}

func RegisterMigration(migration ...string) {
	migrationList = append(migrationList, migration...)
}
