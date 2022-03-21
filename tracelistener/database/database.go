package database

import (
	dbutils "github.com/emerishq/emeris-utils/database"
)

type Instance struct {
	Instance   *dbutils.Instance
	connString string
}

func New(connString string) (*Instance, error) {

	i, err := dbutils.New(connString)

	if err != nil {
		return nil, err
	}

	ii := &Instance{
		Instance:   i,
		connString: connString,
	}

	ii.runMigrations()

	return ii, nil
}

func (i *Instance) Add(query string, data []interface{}) error {
	return i.Instance.Exec(query, data, nil)
}
