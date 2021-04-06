package database

const createDatabase = `
CREATE DATABASE IF NOT EXISTS tracelistener;
`

var migrationList = []string{
	createDatabase,
}

func (i *Instance) runMigrations() {
	for _, m := range migrationList {
		i.d.MustExec(m)
	}
}

func RegisterMigration(migration ...string) {
	migrationList = append(migrationList, migration...)
}
