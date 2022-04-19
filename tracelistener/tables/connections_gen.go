// This file was automatically generated. Please do not edit manually.

package tables

import (
	"fmt"
)

type ConnectionsTable struct {
	tableName string
}

func NewConnectionsTable(tableName string) ConnectionsTable {
	return ConnectionsTable{
		tableName: tableName,
	}
}

func (r ConnectionsTable) CreateTable() string {
	return fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s
		(id serial PRIMARY KEY, height integer NOT NULL, delete_height integer, chain_name text NOT NULL, connection_id text NOT NULL, client_id text NOT NULL, state text NOT NULL, counter_connection_id text NOT NULL, counter_client_id text NOT NULL, UNIQUE (chain_name, connection_id, client_id))
	`, r.tableName)
}

func (r ConnectionsTable) Insert() string {
	return fmt.Sprintf(`
		INSERT INTO %s (height, chain_name, connection_id, client_id, state, counter_connection_id, counter_client_id)
		VALUES (:height, :chain_name, :connection_id, :client_id, :state, :counter_connection_id, :counter_client_id)
	`, r.tableName)
}

func (r ConnectionsTable) Upsert() string {
	return fmt.Sprintf(`
		INSERT INTO %s (height, chain_name, connection_id, client_id, state, counter_connection_id, counter_client_id)
		VALUES (:height, :chain_name, :connection_id, :client_id, :state, :counter_connection_id, :counter_client_id)
		ON CONFLICT (chain_name, connection_id, client_id)
		DO UPDATE
		SET height = EXCLUDED.height, chain_name = EXCLUDED.chain_name, connection_id = EXCLUDED.connection_id, client_id = EXCLUDED.client_id, state = EXCLUDED.state, counter_connection_id = EXCLUDED.counter_connection_id, counter_client_id = EXCLUDED.counter_client_id
	`, r.tableName)
}

func (r ConnectionsTable) Delete() string {
	return fmt.Sprintf(`
		DELETE FROM %s
		WHERE chain_name=:chain_name AND connection_id=:connection_id AND client_id=:client_id
	`, r.tableName)
}
