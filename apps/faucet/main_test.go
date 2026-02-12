package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRateLimiter_allowIP(t *testing.T) {
	r := &rateLimiter{
		ipHits:    make(map[string][]time.Time),
		addrHits:  make(map[string][]time.Time),
		limitIP:   2,
		winIP:     time.Minute,
	}
	if !r.allowIP("1.2.3.4") {
		t.Error("first request should allow")
	}
	if !r.allowIP("1.2.3.4") {
		t.Error("second request should allow")
	}
	if r.allowIP("1.2.3.4") {
		t.Error("third request should rate limit")
	}
	if !r.allowIP("5.6.7.8") {
		t.Error("different IP should allow")
	}
}

func TestRateLimiter_allowAddr(t *testing.T) {
	r := &rateLimiter{
		ipHits:    make(map[string][]time.Time),
		addrHits:  make(map[string][]time.Time),
		limitAddr: 2,
		winAddr:   time.Hour,
	}
	if !r.allowAddr("0x111") {
		t.Error("first addr should allow")
	}
	if !r.allowAddr("0x111") {
		t.Error("second addr should allow")
	}
	if r.allowAddr("0x111") {
		t.Error("third addr should rate limit")
	}
	if !r.allowAddr("0x222") {
		t.Error("different addr should allow")
	}
}

func TestStatusLabel(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{200, "2xx"}, {299, "2xx"}, {404, "4xx"}, {500, "5xx"},
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
	if !strings.Contains(rec.Body.String(), "ok") {
		t.Error("body should contain ok")
	}

	req = httptest.NewRequest(http.MethodPost, "/healthz", nil)
	rec = httptest.NewRecorder()
	handleHealthz(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /healthz = %d, want 405", rec.Code)
	}
}

func TestHandleFaucet(t *testing.T) {
	oldVal := os.Getenv("FORCE_ERROR_RATE")
	os.Unsetenv("FORCE_ERROR_RATE")
	defer func() {
		if oldVal != "" {
			os.Setenv("FORCE_ERROR_RATE", oldVal)
		} else {
			os.Unsetenv("FORCE_ERROR_RATE")
		}
	}()

	limiter := &rateLimiter{
		ipHits:   make(map[string][]time.Time),
		addrHits: make(map[string][]time.Time),
		limitIP:  10,
		limitAddr: 2,
		winIP:    time.Minute,
		winAddr:  time.Hour,
	}
	handler := handleFaucet(limiter)

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/faucet", strings.NewReader(`{"address":"0x123"}`))
		req.RemoteAddr = "1.2.3.4:1234"
		rec := httptest.NewRecorder()
		handler(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("success = %d, want 200", rec.Code)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/faucet", strings.NewReader(`{`))
		req.RemoteAddr = "2.3.4.5:1234"
		rec := httptest.NewRecorder()
		handler(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("invalid JSON = %d, want 400", rec.Code)
		}
	})

	t.Run("empty address", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/faucet", strings.NewReader(`{"address":""}`))
		req.RemoteAddr = "3.4.5.6:1234"
		rec := httptest.NewRecorder()
		handler(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("empty address = %d, want 400", rec.Code)
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/faucet", nil)
		rec := httptest.NewRecorder()
		handler(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("GET = %d, want 405", rec.Code)
		}
	})

	t.Run("X-Forwarded-For malformed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/faucet", strings.NewReader(`{"address":"0xabc"}`))
		req.RemoteAddr = "5.5.5.5:1234"
		req.Header.Set("X-Forwarded-For", ",1.2.3.4") // comma at 0: fall back to RemoteAddr
		rec := httptest.NewRecorder()
		handler(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("malformed X-Forwarded-For = %d, want 200 (uses RemoteAddr)", rec.Code)
		}
	})

	t.Run("rate limit IP", func(t *testing.T) {
		for i := 0; i < 11; i++ {
			req := httptest.NewRequest(http.MethodPost, "/faucet", strings.NewReader(`{"address":"0x`+strconv.Itoa(i)+`"}`))
			req.RemoteAddr = "9.9.9.9:1234"
			rec := httptest.NewRecorder()
			handler(rec, req)
			if i < 10 && rec.Code != http.StatusOK {
				t.Errorf("request %d: %d, want 200", i, rec.Code)
			}
			if i == 10 && rec.Code != http.StatusTooManyRequests {
				t.Errorf("request 10 (rate limit): %d, want 429", rec.Code)
			}
		}
	})

	t.Run("FORCE_ERROR_RATE injects 500", func(t *testing.T) {
		os.Setenv("FORCE_ERROR_RATE", "1.0") // 100% errors
		defer os.Unsetenv("FORCE_ERROR_RATE")
		handlerWithErr := handleFaucet(limiter)
		req := httptest.NewRequest(http.MethodPost, "/faucet", strings.NewReader(`{"address":"0xffff"}`))
		req.RemoteAddr = "8.8.8.8:1234"
		rec := httptest.NewRecorder()
		handlerWithErr(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("FORCE_ERROR_RATE=1.0 = %d, want 500", rec.Code)
		}
	})

	t.Run("body too large", func(t *testing.T) {
		largeBody := strings.Repeat("x", 65*1024) // 65KB exceeds 64KB limit
		req := httptest.NewRequest(http.MethodPost, "/faucet", strings.NewReader(largeBody))
		req.RemoteAddr = "7.7.7.7:1234"
		rec := httptest.NewRecorder()
		handler(rec, req)
		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("body too large = %d, want 413", rec.Code)
		}
	})
}
