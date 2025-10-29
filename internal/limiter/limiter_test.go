package limiter

import (
    "os"
    "sync"
    "testing"
    "time"
)

// mockStorage is a simple in-memory implementation of storage.Storage for tests.
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
        // reset
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

func TestAllowByIP_ExceedAndBlock(t *testing.T) {
    // configure env for limiter
    os.Setenv("MODE", "ip")
    os.Setenv("DEFAULT_LIMIT", "2")
    os.Setenv("DEFAULT_WINDOW", "10") // long window so counts persist during test
    os.Setenv("DEFAULT_BLOCK", "5")

    ms := newMockStorage()
    l := NewLimiter(ms)

    ip := "1.2.3.4"

    // first two should be allowed
    for i := 1; i <= 2; i++ {
        res, err := l.Allow(ip, "")
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if !res.Allowed {
            t.Fatalf("expected allowed on attempt %d, got %+v", i, res)
        }
    }

    // third should be blocked and mark blocked
    res, err := l.Allow(ip, "")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if res.Allowed || !res.Blocked {
        t.Fatalf("expected blocked on third attempt, got %+v", res)
    }

    // subsequent attempt should be immediately blocked
    res2, err := l.Allow(ip, "")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if res2.Allowed || !res2.Blocked {
        t.Fatalf("expected still blocked, got %+v", res2)
    }
}

func TestAllowByToken_OverridesIP(t *testing.T) {
    os.Setenv("MODE", "both")
    os.Setenv("DEFAULT_LIMIT", "1")
    os.Setenv("DEFAULT_WINDOW", "10")
    os.Setenv("DEFAULT_BLOCK", "5")
    os.Setenv("TOKEN_LIMITS", "tok1:3:10:5")

    ms := newMockStorage()
    l := NewLimiter(ms)

    ip := "9.9.9.9"
    token := "tok1"

    // token allows 3
    for i := 1; i <= 3; i++ {
        res, err := l.Allow(ip, token)
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if !res.Allowed {
            t.Fatalf("expected allowed for token attempt %d, got %+v", i, res)
        }
    }

    // 4th should be blocked
    res, err := l.Allow(ip, token)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if res.Allowed || !res.Blocked {
        t.Fatalf("expected blocked for token exceed, got %+v", res)
    }
}

func TestTokenMode_WithoutToken_Denied(t *testing.T) {
    os.Setenv("MODE", "token")
    os.Setenv("DEFAULT_LIMIT", "10")
    os.Setenv("DEFAULT_WINDOW", "1")
    os.Setenv("DEFAULT_BLOCK", "5")

    ms := newMockStorage()
    l := NewLimiter(ms)

    ip := "8.8.8.8"
    res, err := l.Allow(ip, "")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if res.Allowed || !res.Blocked {
        t.Fatalf("in token-only mode without token expected deny+block, got %+v", res)
    }
}
