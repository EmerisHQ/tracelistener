// This file was automatically generated. Please do not edit manually.

package tables

import (
	"fmt"
)

type Cw20BalancesTable struct {
	tableName string
}

func NewCw20BalancesTable(tableName string) Cw20BalancesTable {
	return Cw20BalancesTable{
		tableName: tableName,
	}
}

func (r Cw20BalancesTable) Name() string { return r.tableName }

func (r Cw20BalancesTable) CreateTable() string {
	return fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s
		(id serial PRIMARY KEY NOT NULL, height integer NOT NULL, delete_height integer, chain_name text NOT NULL, contract_address text NOT NULL, address text NOT NULL, amount text NOT NULL, UNIQUE (chain_name, contract_address, address))
	`, r.tableName)
}

func (r Cw20BalancesTable) CreateIndexes() []string {
	return []string{
		
	}
}

func (r Cw20BalancesTable) Migrations() []string {
	return append([]string{r.CreateTable()}, r.CreateIndexes()...)
}

func (r Cw20BalancesTable) Insert() string {
	return fmt.Sprintf(`
		INSERT INTO %s (height, chain_name, contract_address, address, amount)
		VALUES (:height, :chain_name, :contract_address, :address, :amount)
	`, r.tableName)
}

func (r Cw20BalancesTable) Upsert() string {
	return fmt.Sprintf(`
		INSERT INTO %s (height, chain_name, contract_address, address, amount)
		VALUES (:height, :chain_name, :contract_address, :address, :amount)
		ON CONFLICT (chain_name, contract_address, address)
		DO UPDATE
		SET delete_height = NULL, height = EXCLUDED.height, chain_name = EXCLUDED.chain_name, contract_address = EXCLUDED.contract_address, address = EXCLUDED.address, amount = EXCLUDED.amount
		WHERE %s.height < EXCLUDED.height
	`, r.tableName, r.tableName)
}

func (r Cw20BalancesTable) Delete() string {
	return fmt.Sprintf(`
		UPDATE %s
		SET delete_height = :height, height = :height
		WHERE chain_name=:chain_name AND contract_address=:contract_address AND address=:address
		AND delete_height IS NULL
	`, r.tableName)
}
