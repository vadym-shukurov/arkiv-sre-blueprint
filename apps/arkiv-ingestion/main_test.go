package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestSyntheticFetcher(t *testing.T) {
	f := newSyntheticFetcher("1")
	r1, err := f.FetchNext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if r1 == nil {
		t.Fatal("expected record")
	}
	if r1.ChainID != "1" {
		t.Errorf("chainID want 1 got %s", r1.ChainID)
	}
	if r1.IdempotencyKey == "" {
		t.Error("idempotency key required")
	}
	r2, err := f.FetchNext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if r2.IdempotencyKey == r1.IdempotencyKey {
		t.Error("idempotency keys should differ per fetch")
	}
}

func TestStatusLabel(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{200, "2xx"}, {500, "5xx"}, {404, "4xx"},
	}
	for _, tt := range tests {
		if got := statusLabel(tt.code); got != tt.want {
			t.Errorf("statusLabel(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func TestHandleHealthz(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handleHealthz(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("GET /healthz = %d, want 200", rec.Code)
	}
	req = httptest.NewRequest(http.MethodPost, "/healthz", nil)
	rec = httptest.NewRecorder()
	handleHealthz(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /healthz = %d, want 405", rec.Code)
	}
}

func TestIngestWithRetry(t *testing.T) {
	ctx := context.Background()
	r := &IngestRecord{IdempotencyKey: "k1", ChainID: "1", BlockNumber: 1, Data: []byte("{}")}

	// Mock that fails twice then succeeds
	n := 0
	mock := &mockIngester{ingest: func() error {
		n++
		if n < 3 {
			return errMock
		}
		return nil
	}}
	err := ingestWithRetry(ctx, mock, r)
	if err != nil {
		t.Errorf("ingestWithRetry = %v, want nil (succeeds on attempt 3)", err)
	}
	if n != 3 {
		t.Errorf("attempts = %d, want 3", n)
	}
}

func TestIngestWithRetryExhausted(t *testing.T) {
	ctx := context.Background()
	r := &IngestRecord{IdempotencyKey: "k2", ChainID: "1", BlockNumber: 2, Data: []byte("{}")}

	mock := &mockIngester{ingest: func() error { return errMock }}
	err := ingestWithRetry(ctx, mock, r)
	if err != errMock {
		t.Errorf("ingestWithRetry = %v, want errMock", err)
	}
}

func TestIngestWithRetryContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	r := &IngestRecord{IdempotencyKey: "k3", ChainID: "1", BlockNumber: 3, Data: []byte("{}")}

	mock := &mockIngester{ingest: func() error { return errMock }}
	err := ingestWithRetry(ctx, mock, r)
	if err != context.Canceled {
		t.Errorf("ingestWithRetry = %v, want context.Canceled", err)
	}
}

type mockIngester struct {
	ingest func() error
}

func (m *mockIngester) Ingest(ctx context.Context, record IngestRecord) error {
	return m.ingest()
}

var errMock = errors.New("mock")

func TestConfigFromEnv(t *testing.T) {
	os.Setenv("CHAIN_ID", "42")
	os.Setenv("DATABASE_URL", "postgres://a:b@c/d")
	os.Setenv("INGEST_INTERVAL_SEC", "60")
	defer func() {
		os.Unsetenv("CHAIN_ID")
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("INGEST_INTERVAL_SEC")
	}()
	cfg := configFromEnv()
	if cfg.chainID != "42" {
		t.Errorf("chainID = %q, want 42", cfg.chainID)
	}
	if cfg.databaseURL != "postgres://a:b@c/d" {
		t.Errorf("databaseURL = %q", cfg.databaseURL)
	}
	if cfg.interval != 60*time.Second {
		t.Errorf("interval = %v, want 60s", cfg.interval)
	}
}
