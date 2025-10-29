package middleware

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "os"
    "sync"
    "testing"
    "time"

    "github.com/Douglas-Souza40/fctech-rate-limiter/internal/limiter"
)

// mockStorage igual ao usado nos testes do limiter, duplicado aqui para isolamento do pacote
type mockStorage struct {
    mu       sync.Mutex
    counters map[string]struct{
        count int64
        exp   time.Time
    }
    blocked map[string]time.Time
}

func newMockStorage() *mockStorage {
    return &mockStorage{
        counters: make(map[string]struct{count int64; exp time.Time}),
        blocked:  make(map[string]time.Time),
    }
}

func (m *mockStorage) Increment(key string, window time.Duration) (int64, error) {
    m.mu.Lock()
    defer m.mu.Unlock()
    now := time.Now()
    c, ok := m.counters[key]
    if !ok || now.After(c.exp) {
        m.counters[key] = struct{count int64; exp time.Time}{count: 1, exp: now.Add(window)}
        return 1, nil
    }
    c.count++
    m.counters[key] = c
    return c.count, nil
}

func (m *mockStorage) SetBlocked(key string, duration time.Duration) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.blocked[key] = time.Now().Add(duration)
    return nil
}

func (m *mockStorage) IsBlocked(key string) (bool, time.Duration, error) {
    m.mu.Lock()
    defer m.mu.Unlock()
    exp, ok := m.blocked[key]
    if !ok {
        return false, 0, nil
    }
    now := time.Now()
    if now.After(exp) {
        delete(m.blocked, key)
        return false, 0, nil
    }
    return true, exp.Sub(now), nil
}

func TestMiddleware_AllowsUnderLimit(t *testing.T) {
    os.Setenv("MODE", "ip")
    os.Setenv("DEFAULT_LIMIT", "2")
    os.Setenv("DEFAULT_WINDOW", "10")
    os.Setenv("DEFAULT_BLOCK", "5")

    ms := newMockStorage()
    l := limiter.NewLimiter(ms)
    mm := NewLimiterMiddleware(l)

    handler := mm.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("ok"))
    }))

    req := httptest.NewRequest(http.MethodGet, "/ping", nil)
    rr := httptest.NewRecorder()
    handler.ServeHTTP(rr, req)
    if rr.Code != http.StatusOK {
        t.Fatalf("expected 200 first request, got %d", rr.Code)
    }

    // second should also be OK
    rr2 := httptest.NewRecorder()
    handler.ServeHTTP(rr2, req)
    if rr2.Code != http.StatusOK {
        t.Fatalf("expected 200 second request, got %d", rr2.Code)
    }
}

func TestMiddleware_Returns429WhenExceeded(t *testing.T) {
    os.Setenv("MODE", "ip")
    os.Setenv("DEFAULT_LIMIT", "1")
    os.Setenv("DEFAULT_WINDOW", "10")
    os.Setenv("DEFAULT_BLOCK", "5")

    ms := newMockStorage()
    l := limiter.NewLimiter(ms)
    mm := NewLimiterMiddleware(l)

    handler := mm.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("ok"))
    }))

    req := httptest.NewRequest(http.MethodGet, "/ping", nil)
    rr := httptest.NewRecorder()
    handler.ServeHTTP(rr, req)
    if rr.Code != http.StatusOK {
        t.Fatalf("expected 200 first request, got %d", rr.Code)
    }

    // second should be 429
    rr2 := httptest.NewRecorder()
    handler.ServeHTTP(rr2, req)
    if rr2.Code != http.StatusTooManyRequests {
        t.Fatalf("expected 429 second request, got %d", rr2.Code)
    }
    var body map[string]string
    if err := json.Unmarshal(rr2.Body.Bytes(), &body); err != nil {
        t.Fatalf("invalid json body: %v", err)
    }
    if body["message"] == "" {
        t.Fatalf("expected message in body, got empty")
    }
}

func TestMiddleware_UsesTokenOverride(t *testing.T) {
    os.Setenv("MODE", "both")
    os.Setenv("DEFAULT_LIMIT", "1")
    os.Setenv("DEFAULT_WINDOW", "10")
    os.Setenv("DEFAULT_BLOCK", "5")
    os.Setenv("TOKEN_LIMITS", "tok1:2:10:5")

    ms := newMockStorage()
    l := limiter.NewLimiter(ms)
    mm := NewLimiterMiddleware(l)

    handler := mm.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("ok"))
    }))

    req := httptest.NewRequest(http.MethodGet, "/ping", nil)
    req.Header.Set("API_KEY", "tok1")

    // two allowed
    rr1 := httptest.NewRecorder()
    handler.ServeHTTP(rr1, req)
    if rr1.Code != http.StatusOK {
        t.Fatalf("expected 200 first token request, got %d", rr1.Code)
    }
    rr2 := httptest.NewRecorder()
    handler.ServeHTTP(rr2, req)
    if rr2.Code != http.StatusOK {
        t.Fatalf("expected 200 second token request, got %d", rr2.Code)
    }

    // third should be 429
    rr3 := httptest.NewRecorder()
    handler.ServeHTTP(rr3, req)
    if rr3.Code != http.StatusTooManyRequests {
        t.Fatalf("expected 429 third token request, got %d", rr3.Code)
    }
}
