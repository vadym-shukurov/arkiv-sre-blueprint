// Faucet: HTTP API for test tokens. Rate-limited by IP (10/min) and address (2/hr).
// Endpoints: POST /faucet (JSON body: address), GET /healthz, GET /metrics.
package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Rate limit: 10 req/min per IP, 2 req/hour per address.
const (
	perIPLimit    = 10
	perAddrLimit  = 2
	windowPerIP   = time.Minute
	windowPerAddr = time.Hour
)

// FaucetRequest is the JSON body for POST /faucet.
type FaucetRequest struct {
	Address string `json:"address"`
}

// rateLimiter enforces per-IP and per-address limits within sliding windows.
type rateLimiter struct {
	mu        sync.RWMutex
	ipHits    map[string][]time.Time
	addrHits  map[string][]time.Time
	limitIP   int
	limitAddr int
	winIP     time.Duration
	winAddr   time.Duration
}

func (r *rateLimiter) allowIP(ip string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.prune(r.ipHits, ip, r.winIP)
	if len(r.ipHits[ip]) >= r.limitIP {
		return false
	}
	r.ipHits[ip] = append(r.ipHits[ip], time.Now())
	return true
}

func (r *rateLimiter) allowAddr(addr string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.prune(r.addrHits, addr, r.winAddr)
	if len(r.addrHits[addr]) >= r.limitAddr {
		return false
	}
	r.addrHits[addr] = append(r.addrHits[addr], time.Now())
	return true
}

// prune removes timestamps older than the sliding window to keep map size bounded.
func (r *rateLimiter) prune(m map[string][]time.Time, key string, win time.Duration) {
	cutoff := time.Now().Add(-win)
	var valid []time.Time
	for _, t := range m[key] {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	if len(valid) == 0 {
		delete(m, key)
	} else {
		m[key] = valid
	}
}

var (
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "http_requests_total", Help: "Total HTTP requests"},
		[]string{"method", "path", "status"},
	)
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Request duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
	rateLimitHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "faucet_rate_limit_total", Help: "Rate limit hits"},
		[]string{"type"},
	)
)

func init() {
	prometheus.MustRegister(requestsTotal, requestDuration, rateLimitHits)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	limiter := &rateLimiter{
		ipHits:    make(map[string][]time.Time),
		addrHits:  make(map[string][]time.Time),
		limitIP:   perIPLimit,
		limitAddr: perAddrLimit,
		winIP:     windowPerIP,
		winAddr:   windowPerAddr,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handleHealthz)
	mux.HandleFunc("/faucet", handleFaucet(limiter))
	mux.Handle("/metrics", promhttp.Handler())

	addr := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		p = strings.TrimPrefix(p, ":") // allow PORT=8080 or PORT=:8080
		if p != "" {
			addr = ":" + p
		}
	}
	slog.Info("starting", "addr", addr)
	if err := http.ListenAndServe(addr, instrument(mux)); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

// instrument wraps handlers to record Prometheus metrics (method, path, status, duration).
func instrument(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		path := r.URL.Path
		method := r.Method
		ww := &responseWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(ww, r)
		status := statusLabel(ww.status)
		requestsTotal.WithLabelValues(method, path, status).Inc()
		requestDuration.WithLabelValues(method, path).Observe(time.Since(start).Seconds())
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

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func handleFaucet(limiter *rateLimiter) http.HandlerFunc {
	// FORCE_ERROR_RATE (0–1): gameday overlay injects errors to trigger burn-rate alert.
	forceErrorRate := 0.0
	if s := os.Getenv("FORCE_ERROR_RATE"); s != "" {
		if f, err := strconv.ParseFloat(s, 64); err == nil && f >= 0 && f <= 1 {
			forceErrorRate = f
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if forceErrorRate > 0 && rand.Float64() < forceErrorRate {
			http.Error(w, "injected error (gameday)", http.StatusInternalServerError)
			return
		}
		// Use leftmost client IP from X-Forwarded-For (behind proxy); fallback to RemoteAddr.
		ip := r.Header.Get("X-Forwarded-For")
		if idx := strings.Index(ip, ","); idx >= 0 {
			ip = strings.TrimSpace(ip[:idx])
		} else {
			ip = strings.TrimSpace(ip)
		}
		if ip == "" {
			ip, _, _ = strings.Cut(r.RemoteAddr, ":")
		}
		if !limiter.allowIP(ip) {
			rateLimitHits.WithLabelValues("ip").Inc()
			slog.Warn("rate limit ip", "ip", ip)
			http.Error(w, `{"error":"rate limit exceeded (IP)"}`, http.StatusTooManyRequests)
			return
		}
		const maxBodyBytes = 64 * 1024 // 64KB; prevents DoS from huge JSON
		var req FaucetRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
			slog.Warn("invalid body", "err", err)
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				http.Error(w, `{"error":"body too large"}`, http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}
		addr := strings.TrimSpace(req.Address)
		if addr == "" {
			http.Error(w, `{"error":"address required"}`, http.StatusBadRequest)
			return
		}
		if !limiter.allowAddr(addr) {
			rateLimitHits.WithLabelValues("address").Inc()
			slog.Warn("rate limit address", "address", addr)
			http.Error(w, `{"error":"rate limit exceeded (address)"}`, http.StatusTooManyRequests)
			return
		}
		resp := map[string]string{"status": "ok", "address": addr, "tx_hash": "0xstub"}
		body, err := json.Marshal(resp)
		if err != nil {
			slog.Error("encode response", "err", err)
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(body)
		slog.Info("faucet request", "address", addr, "ip", ip)
	}
}
