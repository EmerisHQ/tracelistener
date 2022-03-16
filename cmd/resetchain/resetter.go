package main

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgconn"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type Resetter struct {
	Logger    *zap.SugaredLogger
	DB        *sqlx.DB
	ChainName string
	ChunkSize int
}

func (r Resetter) Reset() error {
	r.Logger.Infow(
		"starting resetter",
		"chainName", r.ChainName,
		"chunkSize", strconv.Itoa(r.ChunkSize),
	)

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
		"liquidity_swaps",
		"liquidity_pools",
	}

	for _, t := range tables {
		l := r.Logger.With("table", t)
		startTime := time.Now()
		l.Info("start")
		err := ResetTable(l, r.DB, t, r.ChainName, r.ChunkSize)
		if err != nil {
			return fmt.Errorf("resetting %s: %w", t, err)
		}
		l.Infow("completed", "took", time.Since(startTime).String())
	}

	return nil
}

type baseQueryParams struct {
	LastId    int    `db:"last_id"`
	ChainName string `db:"chain_name"`
	Limit     int    `db:"limit"`
}

const (
	relationshipNotFoundErrorCode = "42P01"
)

func ResetTable(l *zap.SugaredLogger, db *sqlx.DB, table, chainName string, chunkSize int) error {
	// get last id, we'll use it as a cursor
	row := db.QueryRowx(fmt.Sprintf(`
		SELECT id FROM %s
		WHERE chain_name = $1
		ORDER BY id DESC
		LIMIT 1
	`, table), chainName)
	var lastID int
	err := row.Scan(&lastID)
	if errors.Is(err, sql.ErrNoRows) {
		l.Warn("no rows matched")
		return nil
	}
	if pgerr, ok := err.(*pgconn.PgError); ok && pgerr.Code == relationshipNotFoundErrorCode {
		l.Warn("table doesn't exist")
		return nil
	}
	if err != nil {
		return fmt.Errorf("fetching latest id: %w", err)
	}

	// loop until all rows are deleted
	for {
		rows, err := db.NamedQuery(fmt.Sprintf(`
			DELETE FROM %s
			WHERE id <= :last_id AND chain_name = :chain_name
			ORDER BY id DESC
			LIMIT :limit
			RETURNING id
		`, table), baseQueryParams{
			LastId:    lastID,
			ChainName: chainName,
			Limit:     chunkSize,
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
			return fmt.Errorf("cannot scan row at lastID=%d: %w", lastID, err)
		}

		err = rows.Close()
		if err != nil {
			return fmt.Errorf("closing rows object: %w", err)
		}

		l.Infow("deleted chunk", "lastId", strconv.Itoa(lastID))
	}

	return nil
}
