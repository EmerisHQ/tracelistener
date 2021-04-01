package database

const createDatabase = `
CREATE DATABASE IF NOT EXISTS tracelistener;
`

const createBalancesTable = `
CREATE TABLE IF NOT EXISTS tracelistener.balances (
	address text not null,
	amount integer not null,
	denom text not null,
	height integer not null,
	primary key (address)
)
`

var migrationList = []string{
	createDatabase,
	createBalancesTable,
}

func (i *Instance) runMigrations() {
	for _, m := range migrationList {
		i.d.MustExec(m)
	}
}
