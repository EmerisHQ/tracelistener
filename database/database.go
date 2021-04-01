package database

import (
	"context"
	"fmt"

	"github.com/cockroachdb/cockroach-go/v2/crdb/crdbsqlx"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
)

type Instance struct {
	d *sqlx.DB
}

func New(connString string) (*Instance, error) {
	db, err := sqlx.Connect("pgx", connString)
	if err != nil {
		return nil, err
	}

	i := &Instance{
		d: db,
	}

	i.runMigrations()
	return i, nil
}

func (i *Instance) Add(query string, data []interface{}) error {
	return crdbsqlx.ExecuteTx(context.Background(), i.d, nil, func(tx *sqlx.Tx) error {
		res, err := tx.NamedExec(query, data)
		if err != nil {
			return fmt.Errorf("transaction named exec error, %w", err)
		}

		re, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("transaction named exec error, %w", err)
		}

		if re == 0 {
			return fmt.Errorf("affected rows are zero")
		}

		return nil
	})
}
