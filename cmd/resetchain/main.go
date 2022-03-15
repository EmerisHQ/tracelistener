// resetchain clears CockroachDB data for a specific chain.
// It follows the best practices for performing bulk deletes: https://www.cockroachlabs.com/docs/stable/bulk-delete-data.html
package main

import (
	"flag"
	"log"

	"github.com/allinbits/emeris-utils/logging"

	"github.com/allinbits/emeris-utils/database"
)

func main() {
	logger := logging.New(logging.LoggingConfig{
		JSON: true,
	})

	flags := setupFlag()
	flags.Validate()

	db, err := database.New(flags.db)
	if err != nil {
		logger.Panicw("connecting to DB: %v", err)
	}

	resetter := Resetter{
		Logger:    logger,
		DB:        db.DB,
		ChainName: flags.chain,
		ChunkSize: flags.chunkSize,
	}

	err = resetter.Reset()
	if err != nil {
		logger.Panicw("error: %v", err)
	}
}

type Flags struct {
	db        string
	chain     string
	chunkSize int
}

func (f Flags) Validate() {
	if len(f.db) == 0 {
		flag.Usage()
		log.Fatalf("missing database connection string")
	}

	if len(f.chain) == 0 {
		flag.Usage()
		log.Fatalf("missing chain name")
	}

	if f.chunkSize <= 0 {
		flag.Usage()
		log.Fatalf("chunk size must be greater than 0")
	}
}

func setupFlag() Flags {
	db := flag.String("db", "", "DB connection string, e.g. postgres://root@localhost:27567/tracelistener")
	chain := flag.String("chain", "", "Name of the chain to reset, e.g. cosmos-hub")
	chunkSize := flag.Int("chunk", 10000, "Delete chunk size (default: 10000)")
	flag.Parse()

	return Flags{
		db:        *db,
		chain:     *chain,
		chunkSize: *chunkSize,
	}
}
