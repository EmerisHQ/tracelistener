// This file was automatically generated. Please do not edit manually.

package tables

import (
	"fmt"
)

type ValidatorsTable struct {
	tableName string
}

func NewValidatorsTable(tableName string) ValidatorsTable {
	return ValidatorsTable{
		tableName: tableName,
	}
}

func (r ValidatorsTable) Name() string { return r.tableName }

func (r ValidatorsTable) CreateTable() string {
	return fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s
		(id serial PRIMARY KEY NOT NULL, height integer NOT NULL, delete_height integer, chain_name text NOT NULL, validator_address text NOT NULL, operator_address text NOT NULL, consensus_pubkey_type text, consensus_pubkey_value bytes, jailed bool NOT NULL, status integer NOT NULL, tokens text NOT NULL, delegator_shares text NOT NULL, moniker text, identity text, website text, security_contact text, details text, unbonding_height bigint, unbonding_time text, commission_rate text NOT NULL, max_rate text NOT NULL, max_change_rate text NOT NULL, update_time text NOT NULL, min_self_delegation text NOT NULL, UNIQUE (chain_name, operator_address))
	`, r.tableName)
}

func (r ValidatorsTable) CreateIndexes() []string {
	return []string{
		
	}
}

func (r ValidatorsTable) Migrations() []string {
	return append([]string{r.CreateTable()}, r.CreateIndexes()...)
}

func (r ValidatorsTable) Insert() string {
	return fmt.Sprintf(`
		INSERT INTO %s (height, chain_name, validator_address, operator_address, consensus_pubkey_type, consensus_pubkey_value, jailed, status, tokens, delegator_shares, moniker, identity, website, security_contact, details, unbonding_height, unbonding_time, commission_rate, max_rate, max_change_rate, update_time, min_self_delegation)
		VALUES (:height, :chain_name, :validator_address, :operator_address, :consensus_pubkey_type, :consensus_pubkey_value, :jailed, :status, :tokens, :delegator_shares, :moniker, :identity, :website, :security_contact, :details, :unbonding_height, :unbonding_time, :commission_rate, :max_rate, :max_change_rate, :update_time, :min_self_delegation)
	`, r.tableName)
}

func (r ValidatorsTable) Upsert() string {
	return fmt.Sprintf(`
		INSERT INTO %s (height, chain_name, validator_address, operator_address, consensus_pubkey_type, consensus_pubkey_value, jailed, status, tokens, delegator_shares, moniker, identity, website, security_contact, details, unbonding_height, unbonding_time, commission_rate, max_rate, max_change_rate, update_time, min_self_delegation)
		VALUES (:height, :chain_name, :validator_address, :operator_address, :consensus_pubkey_type, :consensus_pubkey_value, :jailed, :status, :tokens, :delegator_shares, :moniker, :identity, :website, :security_contact, :details, :unbonding_height, :unbonding_time, :commission_rate, :max_rate, :max_change_rate, :update_time, :min_self_delegation)
		ON CONFLICT (chain_name, operator_address)
		DO UPDATE
		SET delete_height = NULL, height = EXCLUDED.height, chain_name = EXCLUDED.chain_name, validator_address = EXCLUDED.validator_address, operator_address = EXCLUDED.operator_address, consensus_pubkey_type = EXCLUDED.consensus_pubkey_type, consensus_pubkey_value = EXCLUDED.consensus_pubkey_value, jailed = EXCLUDED.jailed, status = EXCLUDED.status, tokens = EXCLUDED.tokens, delegator_shares = EXCLUDED.delegator_shares, moniker = EXCLUDED.moniker, identity = EXCLUDED.identity, website = EXCLUDED.website, security_contact = EXCLUDED.security_contact, details = EXCLUDED.details, unbonding_height = EXCLUDED.unbonding_height, unbonding_time = EXCLUDED.unbonding_time, commission_rate = EXCLUDED.commission_rate, max_rate = EXCLUDED.max_rate, max_change_rate = EXCLUDED.max_change_rate, update_time = EXCLUDED.update_time, min_self_delegation = EXCLUDED.min_self_delegation
		WHERE %s.height < EXCLUDED.height
	`, r.tableName, r.tableName)
}

func (r ValidatorsTable) Delete() string {
	return fmt.Sprintf(`
		UPDATE %s
		SET delete_height = :height, height = :height
		WHERE chain_name=:chain_name AND operator_address=:operator_address
		AND delete_height IS NULL
	`, r.tableName)
}
