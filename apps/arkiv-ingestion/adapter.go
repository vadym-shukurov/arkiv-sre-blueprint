package main

import "context"

// ArkivIngester writes ingested records to a backend (e.g. Postgres).
type ArkivIngester interface {
	Ingest(ctx context.Context, record IngestRecord) error
}

// IngestRecord holds chain data for one block; IdempotencyKey deduplicates.
type IngestRecord struct {
	IdempotencyKey string
	ChainID        string
	BlockNumber    uint64
	Data           []byte
}
