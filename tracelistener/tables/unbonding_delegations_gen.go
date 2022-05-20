// This file was automatically generated. Please do not edit manually.

package tables

import (
	"fmt"
)

type UnbondingDelegationsTable struct {
	tableName string
}

func NewUnbondingDelegationsTable(tableName string) UnbondingDelegationsTable {
	return UnbondingDelegationsTable{
		tableName: tableName,
	}
}

func (r UnbondingDelegationsTable) Name() string { return r.tableName }

func (r UnbondingDelegationsTable) CreateTable() string {
	return fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s
		(id serial PRIMARY KEY NOT NULL, height integer NOT NULL, delete_height integer, chain_name text NOT NULL, delegator_address text NOT NULL, validator_address text NOT NULL, entries jsonb NOT NULL, UNIQUE (chain_name, delegator_address, validator_address))
	`, r.tableName)
}

func (r UnbondingDelegationsTable) CreateIndexes() []string {
	return []string{
		
	}
}

func (r UnbondingDelegationsTable) Migrations() []string {
	return append(r.CreateIndexes(), r.CreateTable())
}

func (r UnbondingDelegationsTable) Insert() string {
	return fmt.Sprintf(`
		INSERT INTO %s (height, chain_name, delegator_address, validator_address, entries)
		VALUES (:height, :chain_name, :delegator_address, :validator_address, :entries)
	`, r.tableName)
}

func (r UnbondingDelegationsTable) Upsert() string {
	return fmt.Sprintf(`
		INSERT INTO %s (height, chain_name, delegator_address, validator_address, entries)
		VALUES (:height, :chain_name, :delegator_address, :validator_address, :entries)
		ON CONFLICT (chain_name, delegator_address, validator_address)
		DO UPDATE
		SET delete_height = NULL, height = EXCLUDED.height, chain_name = EXCLUDED.chain_name, delegator_address = EXCLUDED.delegator_address, validator_address = EXCLUDED.validator_address, entries = EXCLUDED.entries
		WHERE %s.height < EXCLUDED.height
	`, r.tableName, r.tableName)
}

func (r UnbondingDelegationsTable) Delete() string {
	return fmt.Sprintf(`
		UPDATE %s
		SET delete_height = :height, height = :height
		WHERE chain_name=:chain_name AND delegator_address=:delegator_address AND validator_address=:validator_address
		AND delete_height IS NULL
	`, r.tableName)
}
