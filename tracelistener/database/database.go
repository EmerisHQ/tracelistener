package database

import (
	dbutils "github.com/allinbits/demeris-backend/utils/database"
)

type Instance struct {
	d          *dbutils.Instance
	connString string
}

func New(connString string) (*Instance, error) {

	i, err := dbutils.New(connString)

	if err != nil {
		return nil, err
	}

	ii := &Instance{
		d:          i,
		connString: connString,
	}

	ii.runMigrations()

	return ii, nil
}

func (i *Instance) Add(query string, data []interface{}) error {
	return i.d.Exec(query, data, nil)
}
