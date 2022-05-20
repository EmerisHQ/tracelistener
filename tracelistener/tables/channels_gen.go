// This file was automatically generated. Please do not edit manually.

package tables

import (
	"fmt"
)

type ChannelsTable struct {
	tableName string
}

func NewChannelsTable(tableName string) ChannelsTable {
	return ChannelsTable{
		tableName: tableName,
	}
}

func (r ChannelsTable) Name() string { return r.tableName }

func (r ChannelsTable) CreateTable() string {
	return fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s
		(id serial PRIMARY KEY NOT NULL, height integer NOT NULL, delete_height integer, chain_name text NOT NULL, channel_id text NOT NULL, counter_channel_id text NOT NULL, port text NOT NULL, state integer NOT NULL, hops text[] NOT NULL, UNIQUE (chain_name, channel_id, port))
	`, r.tableName)
}

func (r ChannelsTable) CreateIndexes() []string {
	return []string{
		
	}
}

func (r ChannelsTable) Migrations() []string {
	return append(r.CreateIndexes(), r.CreateTable())
}

func (r ChannelsTable) Insert() string {
	return fmt.Sprintf(`
		INSERT INTO %s (height, chain_name, channel_id, counter_channel_id, port, state, hops)
		VALUES (:height, :chain_name, :channel_id, :counter_channel_id, :port, :state, :hops)
	`, r.tableName)
}

func (r ChannelsTable) Upsert() string {
	return fmt.Sprintf(`
		INSERT INTO %s (height, chain_name, channel_id, counter_channel_id, port, state, hops)
		VALUES (:height, :chain_name, :channel_id, :counter_channel_id, :port, :state, :hops)
		ON CONFLICT (chain_name, channel_id, port)
		DO UPDATE
		SET delete_height = NULL, height = EXCLUDED.height, chain_name = EXCLUDED.chain_name, channel_id = EXCLUDED.channel_id, counter_channel_id = EXCLUDED.counter_channel_id, port = EXCLUDED.port, state = EXCLUDED.state, hops = EXCLUDED.hops
		WHERE %s.height < EXCLUDED.height
	`, r.tableName, r.tableName)
}

func (r ChannelsTable) Delete() string {
	return fmt.Sprintf(`
		UPDATE %s
		SET delete_height = :height, height = :height
		WHERE chain_name=:chain_name AND channel_id=:channel_id AND port=:port
		AND delete_height IS NULL
	`, r.tableName)
}
