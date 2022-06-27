// resetchain clears CockroachDB data for a specific chain.
// It follows the best practices for performing bulk deletes: https://www.cockroachlabs.com/docs/stable/bulk-delete-data.html
package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/emerishq/tracelistener/database"
	"github.com/emerishq/tracelistener/logging"
)

func main() {
	logger := logging.New(logging.LoggingConfig{
		JSON: true,
	})

	flags := setupFlag()
	err := flags.Validate()
	if err != nil {
		flag.Usage()
		logger.Panic("invalid flags", err)
	}

	db, err := database.New(flags.db)
	if err != nil {
		logger.Panic("connecting to DB", err)
	}

	db.DB.SetMaxOpenConns(10)
	db.DB.SetMaxIdleConns(10)

	tables := strings.Split(flags.tables, ",")
	resetter := Resetter{
		Logger:    logger,
		DB:        db.DB,
		ChainName: flags.chain,
		ChunkSize: flags.chunkSize,
		Tables:    tables,
	}

	err = resetter.Reset()
	if err != nil {
		logger.Panic("error", err)
	}
}

type Flags struct {
	db        string
	chain     string
	chunkSize int
	tables    string
}

func (f Flags) Validate() error {
	if len(f.db) == 0 {
		return fmt.Errorf("missing database connection string")
	}

	if len(f.chain) == 0 {
		return fmt.Errorf("missing chain name")
	}

	if f.chunkSize <= 0 {
		return fmt.Errorf("chunk size must be greater than 0")
	}

	if len(f.tables) == 0 {
		return fmt.Errorf("missing tables to reset")
	}

	return nil
}

var defaultTables = []string{
	"balances@balances_chain_name_id_idx",
	"connections@connections_chain_name_id_idx",
	"delegations@delegations_chain_name_id_idx",
	"unbonding_delegations@unbonding_delegations_chain_name_id_idx",
	"denom_traces@denom_traces_chain_name_id_idx",
	"channels@channels_chain_name_id_idx",
	"auth@auth_chain_name_id_idx",
	"clients@clients_chain_name_id_idx",
	"validators@validators_chain_name_id_idx",
	"liquidity_swaps@liquidity_swaps_chain_name_id_idx",
	"liquidity_pools@liquidity_pools_chain_name_id_idx",
}

func setupFlag() Flags {
	db := flag.String("db", "", "DB connection string, e.g. postgres://root@localhost:26257/tracelistener")
	chain := flag.String("chain", "", "Name of the chain to reset, e.g. cosmos-hub")
	chunkSize := flag.Int("chunk", 5000, "Delete chunk size (default: 5000)")
	tables := flag.String("tables", strings.Join(defaultTables, ","), "Comma separated list of tables to reset. If not specified, all tables will be reset.")
	flag.Parse()

	return Flags{
		db:        *db,
		chain:     *chain,
		chunkSize: *chunkSize,
		tables:    *tables,
	}
}
