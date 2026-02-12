// Arkiv-ingestion: Fetches chain data and ingests into Postgres. Runs a ticker-based worker.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	ingestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "arkiv_ingest_total", Help: "Ingestion attempts"},
		[]string{"status"},
	)
	ingestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "arkiv_ingest_duration_seconds", Help: "Ingest latency", Buckets: prometheus.DefBuckets},
		[]string{"status"},
	)
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "http_requests_total", Help: "HTTP requests"},
		[]string{"method", "path", "status"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "http_request_duration_seconds", Help: "Request latency", Buckets: prometheus.DefBuckets},
		[]string{"method", "path"},
	)
)

func init() {
	prometheus.MustRegister(ingestTotal, ingestDuration, httpRequestsTotal, httpRequestDuration)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := configFromEnv()
	ingester, err := newPostgresIngester(context.Background(), cfg.databaseURL)
	if err != nil {
		slog.Error("create ingester", "err", err)
		os.Exit(1)
	}

	fetcher := newSyntheticFetcher(cfg.chainID)
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go runWorker(ctx, ingester, fetcher, cfg.interval, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handleHealthz)
	mux.Handle("/metrics", promhttp.Handler())

	addr := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		p = strings.TrimPrefix(p, ":") // allow PORT=8080 or PORT=:8080
		if p != "" {
			addr = ":" + p
		}
	}

	// Use http.Server for graceful shutdown on SIGTERM/SIGINT.
	srv := &http.Server{Addr: addr, Handler: instrument(mux)}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server stopped", "err", err)
			cancel() // trigger shutdown so main can exit
		}
	}()
	slog.Info("starting", "addr", addr)

	<-ctx.Done()
	slog.Info("shutting down")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown", "err", err)
	}
}

// runWorker fetches records at interval and ingests them; exits on ctx.Done().
func runWorker(ctx context.Context, ingester ArkivIngester, fetcher *syntheticFetcher, interval time.Duration, log *slog.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			record, err := fetcher.FetchNext(ctx)
			if err != nil {
				log.Warn("fetch failed", "err", err)
				ingestTotal.WithLabelValues("error").Inc()
				continue
			}
			if record == nil {
				continue
			}
			start := time.Now()
			ingestErr := ingestWithRetry(ctx, ingester, record)
			duration := time.Since(start).Seconds()
			status := "ok"
			if ingestErr != nil {
				status = "error"
				log.Warn("ingest failed", "key", record.IdempotencyKey, "err", ingestErr)
			}
			ingestTotal.WithLabelValues(status).Inc()
			ingestDuration.WithLabelValues(status).Observe(duration)
		}
	}
}

// ingestWithRetry tries up to 3 times with exponential backoff (1s, 2s, 3s).
func ingestWithRetry(ctx context.Context, ingester ArkivIngester, r *IngestRecord) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		err := ingester.Ingest(ctx, *r)
		if err == nil {
			return nil
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt+1) * time.Second):
		}
	}
	return lastErr
}

// config holds env-derived settings. DATABASE_URL must be set for real deployments.
type config struct {
	databaseURL string
	chainID     string
	interval    time.Duration
}

// configFromEnv reads settings from env. REDACTED default is for tests only; real runs need DATABASE_URL.
func configFromEnv() config {
	pg := os.Getenv("DATABASE_URL")
	if pg == "" {
		pg = "postgres://postgres:REDACTED@localhost:5432/arkiv?sslmode=disable"
	}
	interval := 30 * time.Second
	if s := os.Getenv("INGEST_INTERVAL_SEC"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			interval = time.Duration(n) * time.Second
		}
	}
	chainID := os.Getenv("CHAIN_ID")
	if chainID == "" {
		chainID = "1"
	}
	return config{
		databaseURL: pg,
		chainID:     chainID,
		interval:    interval,
	}
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// instrument wraps handlers to record Prometheus metrics.
func instrument(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		path := r.URL.Path
		method := r.Method
		ww := &responseWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(ww, r)
		status := statusLabel(ww.status)
		httpRequestsTotal.WithLabelValues(method, path, status).Inc()
		httpRequestDuration.WithLabelValues(method, path).Observe(time.Since(start).Seconds())
	})
}

// responseWriter captures status code for Prometheus labeling.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func statusLabel(code int) string {
	switch {
	case code >= 500:
		return "5xx"
	case code >= 400:
		return "4xx"
	case code >= 300:
		return "3xx"
	case code >= 200:
		return "2xx"
	default:
		return "unknown"
	}
}
