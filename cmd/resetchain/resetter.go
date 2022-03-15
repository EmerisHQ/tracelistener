package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
)

type Resetter struct {
	DB        *sqlx.DB
	ChainName string
	ChunkSize int
}

func (r Resetter) Reset() error {
	log.Printf("starting resetter (chain=%s, chunkSize=%d)", r.ChainName, r.ChunkSize)

	tables := []string{
		"balances",
		"connections",
		"delegations",
		"unbonding_delegations",
		"denom_traces",
		"channels",
		"auth",
		"clients",
		"validators",
	}

	for _, t := range tables {
		startTime := time.Now()
		log.Printf("[%s] start", t)
		if err := r.ResetTable(t); err != nil {
			return fmt.Errorf("resetting %s: %w", t, err)
		}
		endTime := time.Now()
		log.Printf("[%s] completed (took %s)", t, endTime.Sub(startTime))
	}

	return nil
}

type baseQueryParams struct {
	LastId    int    `db:"last_id"`
	ChainName string `db:"chain_name"`
	Limit     int    `db:"limit"`
}

func (r Resetter) ResetTable(table string) error {
	// get last id, we'll use it as a cursor
	row := r.DB.QueryRowx(fmt.Sprintf(`
		SELECT id FROM %s
		WHERE chain_name = $1
		ORDER BY id DESC
		LIMIT 1
	`, table), r.ChainName)
	var lastID int
	err := row.Scan(&lastID)
	if err == sql.ErrNoRows {
		log.Printf("[%s] no rows matched", table)
		return nil
	}
	if err != nil {
		return fmt.Errorf("fetching latest id: %w", err)
	}

	// loop until all rows are deleted
	for {
		rows, err := r.DB.NamedQuery(fmt.Sprintf(`
			DELETE FROM %s
			WHERE id <= :last_id AND chain_name = :chain_name
			ORDER BY id DESC
			LIMIT :limit
			RETURNING id
		`, table), baseQueryParams{
			LastId:    lastID,
			ChainName: r.ChainName,
			Limit:     r.ChunkSize,
		})
		if err != nil {
			return err
		}

		cont := rows.Next()
		if !cont {
			break
		}

		err = rows.Scan(&lastID)
		if err != nil {
			return fmt.Errorf("deleting data: %w", err)
		}

		log.Printf("[%s] deleted chunk (lastID=%d)", table, lastID)
	}

	return nil
}
