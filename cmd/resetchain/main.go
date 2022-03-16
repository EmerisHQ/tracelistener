// resetchain clears CockroachDB data for a specific chain.
// It follows the best practices for performing bulk deletes: https://www.cockroachlabs.com/docs/stable/bulk-delete-data.html
package main

import (
	"flag"
	"fmt"

	"github.com/allinbits/emeris-utils/database"
	"github.com/allinbits/emeris-utils/logging"
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

	resetter := Resetter{
		Logger:    logger,
		DB:        db.DB,
		ChainName: flags.chain,
		ChunkSize: flags.chunkSize,
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

	return nil
}

func setupFlag() Flags {
	db := flag.String("db", "", "DB connection string, e.g. postgres://root@localhost:26257/tracelistener")
	chain := flag.String("chain", "", "Name of the chain to reset, e.g. cosmos-hub")
	chunkSize := flag.Int("chunk", 5000, "Delete chunk size (default: 5000)")
	flag.Parse()

	return Flags{
		db:        *db,
		chain:     *chain,
		chunkSize: *chunkSize,
	}
}
