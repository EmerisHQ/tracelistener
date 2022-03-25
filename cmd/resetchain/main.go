// resetchain clears CockroachDB data for a specific chain.
// It follows the best practices for performing bulk deletes: https://www.cockroachlabs.com/docs/stable/bulk-delete-data.html
package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/emerishq/emeris-utils/database"
	"github.com/emerishq/emeris-utils/logging"
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

	tables := strings.Split(flags.forceIndexes, ",")
	resetter := Resetter{
		Logger:    logger,
		DB:        db.DB,
		ChainName: flags.chain,
		ChunkSize: flags.chunkSize,
		Tables:    GetTables(tables, flags.forceIndexes),
	}

	err = resetter.Reset()
	if err != nil {
		logger.Panic("error", err)
	}
}

func GetTables(tables []string, forceIndexesFlag string) []string {
	overrides := getOverrideTableMap(forceIndexesFlag)
	return applyOverride(tables, overrides)
}

func getOverrideTableMap(forceIndexesFlag string) map[string]string {
	overrides := make(map[string]string)
	if len(forceIndexesFlag) > 0 {
		forceIndexes := strings.Split(forceIndexesFlag, ",")
		for _, forceIndex := range forceIndexes {
			tableIndex := strings.Split(forceIndex, "@")
			overrides[tableIndex[0]] = forceIndex
		}
	}
	return overrides
}

func applyOverride(base []string, overrides map[string]string) []string {
	res := make([]string, 0, len(base))

	// apply overrides
	for _, t := range base {
		if override, ok := overrides[t]; ok {
			res = append(res, override)
		} else {
			res = append(res, t)
		}
	}

	return res
}

type Flags struct {
	db           string
	chain        string
	chunkSize    int
	forceIndexes string
	tables       string
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
	"balances",
	"connections",
	"delegations",
	"unbonding_delegations",
	"denom_traces",
	"channels",
	"auth",
	"clients",
	"validators",
	"liquidity_swaps",
	"liquidity_pools",
}

func setupFlag() Flags {
	db := flag.String("db", "", "DB connection string, e.g. postgres://root@localhost:26257/tracelistener")
	chain := flag.String("chain", "", "Name of the chain to reset, e.g. cosmos-hub")
	chunkSize := flag.Int("chunk", 5000, "Delete chunk size (default: 5000)")
	forceIndexes := flag.String("force-indexes", "", "Comma separated list of \"table@index\" elements to force the use of a certain database index. E.g. auth@some_idx,balances@other_idx")
	tables := flag.String("tables", strings.Join(defaultTables, ","), "Comma separated list of tables to reset. If not specified, all tables will be reset.")
	flag.Parse()

	return Flags{
		db:           *db,
		chain:        *chain,
		chunkSize:    *chunkSize,
		forceIndexes: *forceIndexes,
		tables:       *tables,
	}
}
