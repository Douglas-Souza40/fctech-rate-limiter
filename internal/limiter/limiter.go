package limiter

import (
    "fmt"
    "os"
    "strconv"
    "strings"
    "time"

    "github.com/Douglas-Souza40/fctech-rate-limiter/internal/storage"
)

type TokenConfig struct {
    Limit  int
    Window time.Duration
    Block  time.Duration
}

type Limiter struct {
    store storage.Storage
    mode  string // ip | token | both

    defaultLimit  int
    defaultWindow time.Duration
    defaultBlock  time.Duration

    tokenConfigs map[string]TokenConfig
}

// NewLimiter constructs a limiter reading environment variables for defaults.
func NewLimiter(store storage.Storage) *Limiter {
    l := &Limiter{
        store:        store,
        mode:         getEnv("MODE", "both"),
        defaultLimit: getEnvAsInt("DEFAULT_LIMIT", 10),
        defaultWindow: time.Duration(getEnvAsInt("DEFAULT_WINDOW", 1)) * time.Second,
        defaultBlock:  time.Duration(getEnvAsInt("DEFAULT_BLOCK", 300)) * time.Second,
        tokenConfigs:  parseTokenConfigs(getEnv("TOKEN_LIMITS", "")),
    }
    return l
}

func getEnv(key, fallback string) string {
    v := os.Getenv(key)
    if v == "" {
        return fallback
    }
    return v
}

func getEnvAsInt(key string, fallback int) int {
    v := os.Getenv(key)
    if v == "" {
        return fallback
    }
    i, err := strconv.Atoi(v)
    if err != nil {
        return fallback
    }
    return i
}

// TOKEN_LIMITS format: token:limit:window:block,token2:...
func parseTokenConfigs(raw string) map[string]TokenConfig {
    out := map[string]TokenConfig{}
    if strings.TrimSpace(raw) == "" {
        return out
    }
    parts := strings.Split(raw, ",")
    for _, p := range parts {
        p = strings.TrimSpace(p)
        if p == "" {
            continue
        }
        seg := strings.Split(p, ":")
        if len(seg) < 2 {
            continue
        }
        token := seg[0]
        limit := 0
        window := 1
        block := 300
        if v, err := strconv.Atoi(seg[1]); err == nil {
            limit = v
        }
        if len(seg) >= 3 {
            if v, err := strconv.Atoi(seg[2]); err == nil {
                window = v
            }
        }
        if len(seg) >= 4 {
            if v, err := strconv.Atoi(seg[3]); err == nil {
                block = v
            }
        }
        out[token] = TokenConfig{Limit: limit, Window: time.Duration(window) * time.Second, Block: time.Duration(block) * time.Second}
    }
    return out
}

type AllowResult struct {
    Allowed     bool
    Count       int64
    Limit       int
    Blocked     bool
    BlockRemain time.Duration
}

// Allow checks whether a request for given ip and apiKey is allowed. If apiKey is non-empty
// and a token config exists, token config overrides IP limits.
func (l *Limiter) Allow(ip string, apiKey string) (AllowResult, error) {
    // decide strategy
    useToken := false
    var cfg TokenConfig
    if apiKey != "" {
        if c, ok := l.tokenConfigs[apiKey]; ok {
            useToken = true
            cfg = c
        }
    }

    var key string
    var limit int
    var window time.Duration
    var block time.Duration

    if useToken {
        key = fmt.Sprintf("token:%s", apiKey)
        limit = cfg.Limit
        window = cfg.Window
        block = cfg.Block
    } else if l.mode == "token" {
        // if mode is token-only and no token present, use default deny by setting limit 0
        key = fmt.Sprintf("ip:%s", ip)
        limit = 0
        window = l.defaultWindow
        block = l.defaultBlock
    } else {
        key = fmt.Sprintf("ip:%s", ip)
        limit = l.defaultLimit
        window = l.defaultWindow
        block = l.defaultBlock
    }

    // check blocked
    blocked, rem, err := l.store.IsBlocked(key)
    if err != nil {
        return AllowResult{}, err
    }
    if blocked {
        return AllowResult{Allowed: false, Blocked: true, BlockRemain: rem}, nil
    }

    // if limit is 0, disallow
    if limit <= 0 {
        // set block and return
        _ = l.store.SetBlocked(key, block)
        return AllowResult{Allowed: false, Limit: limit, Count: 0, Blocked: true, BlockRemain: block}, nil
    }

    cnt, err := l.store.Increment(key, window)
    if err != nil {
        return AllowResult{}, err
    }
    if int(cnt) > limit {
        // exceed -> block
        _ = l.store.SetBlocked(key, block)
        return AllowResult{Allowed: false, Count: cnt, Limit: limit, Blocked: true, BlockRemain: block}, nil
    }

    return AllowResult{Allowed: true, Count: cnt, Limit: limit, Blocked: false}, nil
}
