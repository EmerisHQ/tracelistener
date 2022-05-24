// This file was automatically generated. Please do not edit manually.

package tables

import (
	"fmt"
)

type ClientsTable struct {
	tableName string
}

func NewClientsTable(tableName string) ClientsTable {
	return ClientsTable{
		tableName: tableName,
	}
}

func (r ClientsTable) Name() string { return r.tableName }

func (r ClientsTable) CreateTable() string {
	return fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s
		(id serial PRIMARY KEY NOT NULL, height integer NOT NULL, delete_height integer, chain_name text NOT NULL, chain_id text NOT NULL, client_id text NOT NULL, latest_height numeric NOT NULL, trusting_period numeric NOT NULL, UNIQUE (chain_name, chain_id, client_id))
	`, r.tableName)
}

func (r ClientsTable) CreateIndexes() []string {
	return []string{
		
	}
}

func (r ClientsTable) Migrations() []string {
	return append([]string{r.CreateTable()}, r.CreateIndexes()...)
}

func (r ClientsTable) Insert() string {
	return fmt.Sprintf(`
		INSERT INTO %s (height, chain_name, chain_id, client_id, latest_height, trusting_period)
		VALUES (:height, :chain_name, :chain_id, :client_id, :latest_height, :trusting_period)
	`, r.tableName)
}

func (r ClientsTable) Upsert() string {
	return fmt.Sprintf(`
		INSERT INTO %s (height, chain_name, chain_id, client_id, latest_height, trusting_period)
		VALUES (:height, :chain_name, :chain_id, :client_id, :latest_height, :trusting_period)
		ON CONFLICT (chain_name, chain_id, client_id)
		DO UPDATE
		SET delete_height = NULL, height = EXCLUDED.height, chain_name = EXCLUDED.chain_name, chain_id = EXCLUDED.chain_id, client_id = EXCLUDED.client_id, latest_height = EXCLUDED.latest_height, trusting_period = EXCLUDED.trusting_period
		WHERE %s.height < EXCLUDED.height
	`, r.tableName, r.tableName)
}

func (r ClientsTable) Delete() string {
	return fmt.Sprintf(`
		UPDATE %s
		SET delete_height = :height, height = :height
		WHERE chain_name=:chain_name AND chain_id=:chain_id AND client_id=:client_id
		AND delete_height IS NULL
	`, r.tableName)
}
