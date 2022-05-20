// This file was automatically generated. Please do not edit manually.

package tables

import (
	"fmt"
)

type AuthTable struct {
	tableName string
}

func NewAuthTable(tableName string) AuthTable {
	return AuthTable{
		tableName: tableName,
	}
}

func (r AuthTable) Name() string { return r.tableName }

func (r AuthTable) CreateTable() string {
	return fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s
		(id serial PRIMARY KEY NOT NULL, height integer NOT NULL, delete_height integer, chain_name text NOT NULL, address text NOT NULL, sequence_number numeric NOT NULL, account_number numeric NOT NULL, UNIQUE (chain_name, address, account_number))
	`, r.tableName)
}

func (r AuthTable) CreateIndexes() []string {
	return []string{
		
	}
}

func (r AuthTable) Migrations() []string {
	return append(r.CreateIndexes(), r.CreateTable())
}

func (r AuthTable) Insert() string {
	return fmt.Sprintf(`
		INSERT INTO %s (height, chain_name, address, sequence_number, account_number)
		VALUES (:height, :chain_name, :address, :sequence_number, :account_number)
	`, r.tableName)
}

func (r AuthTable) Upsert() string {
	return fmt.Sprintf(`
		INSERT INTO %s (height, chain_name, address, sequence_number, account_number)
		VALUES (:height, :chain_name, :address, :sequence_number, :account_number)
		ON CONFLICT (chain_name, address, account_number)
		DO UPDATE
		SET delete_height = NULL, height = EXCLUDED.height, chain_name = EXCLUDED.chain_name, address = EXCLUDED.address, sequence_number = EXCLUDED.sequence_number, account_number = EXCLUDED.account_number
		WHERE %s.height < EXCLUDED.height
	`, r.tableName, r.tableName)
}

func (r AuthTable) Delete() string {
	return fmt.Sprintf(`
		UPDATE %s
		SET delete_height = :height, height = :height
		WHERE chain_name=:chain_name AND address=:address AND account_number=:account_number
		AND delete_height IS NULL
	`, r.tableName)
}
