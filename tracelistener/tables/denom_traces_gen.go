// This file was automatically generated. Please do not edit manually.

package tables

import (
	"fmt"
)

type DenomTracesTable struct {
	tableName string
}

func NewDenomTracesTable(tableName string) DenomTracesTable {
	return DenomTracesTable{
		tableName: tableName,
	}
}

func (r DenomTracesTable) CreateTable() string {
	return fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s
		(id serial PRIMARY KEY,height integer NOT NULL,delete_height integer,chain_name text NOT NULL,path text NOT NULL,base_denom text NOT NULL,hash text NOT NULL,UNIQUE (chain_name,hash))
	`, r.tableName)
}

func (r DenomTracesTable) Insert() string {
	return fmt.Sprintf(`
		INSERT INTO %s (height,chain_name,path,base_denom,hash)
		VALUES (:height,:chain_name,:path,:base_denom,:hash)
	`, r.tableName)
}

func (r DenomTracesTable) Upsert() string {
	return fmt.Sprintf(`
		INSERT INTO %s (height,chain_name,path,base_denom,hash)
		VALUES (:height,:chain_name,:path,:base_denom,:hash)
		ON CONFLICT (chain_name,hash)
		DO UPDATE
		SET height = EXCLUDED.height,chain_name = EXCLUDED.chain_name,path = EXCLUDED.path,base_denom = EXCLUDED.base_denom,hash = EXCLUDED.hash
	`, r.tableName)
}

func (r DenomTracesTable) Delete() string {
	return fmt.Sprintf(`
		DELETE FROM %s
		WHERE chain_name=:chain_name AND hash=:hash
	`, r.tableName)
}
