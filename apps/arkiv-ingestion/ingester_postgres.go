package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// postgresIngester writes to ingestion_records.
// Uses ON CONFLICT (idempotency_key) DO NOTHING for idempotency and deduplication.
type postgresIngester struct {
	pool *pgxpool.Pool
}

func newPostgresIngester(ctx context.Context, connStr string) (*postgresIngester, error) {
	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS ingestion_records (
			idempotency_key TEXT PRIMARY KEY,
			chain_id TEXT,
			block_number BIGINT,
			data JSONB,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("create table: %w", err)
	}
	return &postgresIngester{pool: pool}, nil
}

func (p *postgresIngester) Ingest(ctx context.Context, r IngestRecord) error {
	_, err := p.pool.Exec(ctx,
		`INSERT INTO ingestion_records (idempotency_key, chain_id, block_number, data)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (idempotency_key) DO NOTHING`,
		r.IdempotencyKey, r.ChainID, r.BlockNumber, json.RawMessage(r.Data),
	)
	return err
}
