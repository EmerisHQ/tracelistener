// This file was automatically generated. Please do not edit manually.

package tables

import (
	"fmt"
)

type DelegationsTable struct {
	tableName string
}

func NewDelegationsTable(tableName string) DelegationsTable {
	return DelegationsTable{
		tableName: tableName,
	}
}

func (r DelegationsTable) Name() string { return r.tableName }

func (r DelegationsTable) CreateTable() string {
	return fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s
		(id serial PRIMARY KEY NOT NULL, height integer NOT NULL, delete_height integer, chain_name text NOT NULL, delegator_address text NOT NULL, validator_address text NOT NULL, amount text NOT NULL, UNIQUE (chain_name, delegator_address, validator_address))
	`, r.tableName)
}

func (r DelegationsTable) Insert() string {
	return fmt.Sprintf(`
		INSERT INTO %s (height, chain_name, delegator_address, validator_address, amount)
		VALUES (:height, :chain_name, :delegator_address, :validator_address, :amount)
	`, r.tableName)
}

func (r DelegationsTable) Upsert() string {
	return fmt.Sprintf(`
		INSERT INTO %s (height, chain_name, delegator_address, validator_address, amount)
		VALUES (:height, :chain_name, :delegator_address, :validator_address, :amount)
		ON CONFLICT (chain_name, delegator_address, validator_address)
		DO UPDATE
		SET delete_height = NULL, height = EXCLUDED.height, chain_name = EXCLUDED.chain_name, delegator_address = EXCLUDED.delegator_address, validator_address = EXCLUDED.validator_address, amount = EXCLUDED.amount
		WHERE %s.height < EXCLUDED.height
	`, r.tableName, r.tableName)
}

func (r DelegationsTable) Delete() string {
	return fmt.Sprintf(`
		UPDATE %s
		SET delete_height = :height, height = :height
		WHERE chain_name=:chain_name AND delegator_address=:delegator_address AND validator_address=:validator_address
		AND delete_height IS NULL
	`, r.tableName)
}
