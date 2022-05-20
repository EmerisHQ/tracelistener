// This file was automatically generated. Please do not edit manually.

package tables

import (
	"fmt"
)

type Cw20TokenInfoTable struct {
	tableName string
}

func NewCw20TokenInfoTable(tableName string) Cw20TokenInfoTable {
	return Cw20TokenInfoTable{
		tableName: tableName,
	}
}

func (r Cw20TokenInfoTable) Name() string { return r.tableName }

func (r Cw20TokenInfoTable) CreateTable() string {
	return fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s
		(id serial PRIMARY KEY NOT NULL, height integer NOT NULL, delete_height integer, chain_name text NOT NULL, contract_address text NOT NULL, name text NOT NULL, symbol text NOT NULL, decimals integer NOT NULL, total_supply text NOT NULL, UNIQUE (chain_name, contract_address))
	`, r.tableName)
}

func (r Cw20TokenInfoTable) CreateIndexes() []string {
	return []string{
		
	}
}

func (r Cw20TokenInfoTable) Migrations() []string {
	return append(r.CreateIndexes(), r.CreateTable())
}

func (r Cw20TokenInfoTable) Insert() string {
	return fmt.Sprintf(`
		INSERT INTO %s (height, chain_name, contract_address, name, symbol, decimals, total_supply)
		VALUES (:height, :chain_name, :contract_address, :name, :symbol, :decimals, :total_supply)
	`, r.tableName)
}

func (r Cw20TokenInfoTable) Upsert() string {
	return fmt.Sprintf(`
		INSERT INTO %s (height, chain_name, contract_address, name, symbol, decimals, total_supply)
		VALUES (:height, :chain_name, :contract_address, :name, :symbol, :decimals, :total_supply)
		ON CONFLICT (chain_name, contract_address)
		DO UPDATE
		SET delete_height = NULL, height = EXCLUDED.height, chain_name = EXCLUDED.chain_name, contract_address = EXCLUDED.contract_address, name = EXCLUDED.name, symbol = EXCLUDED.symbol, decimals = EXCLUDED.decimals, total_supply = EXCLUDED.total_supply
		WHERE %s.height < EXCLUDED.height
	`, r.tableName, r.tableName)
}

func (r Cw20TokenInfoTable) Delete() string {
	return fmt.Sprintf(`
		UPDATE %s
		SET delete_height = :height, height = :height
		WHERE chain_name=:chain_name AND contract_address=:contract_address
		AND delete_height IS NULL
	`, r.tableName)
}
