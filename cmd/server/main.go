package main

import (
    "fmt"
    "log"
    "net/http"
    "os"

    "github.com/Douglas-Souza40/fctech-rate-limiter/internal/limiter"
    "github.com/Douglas-Souza40/fctech-rate-limiter/internal/storage"
    "github.com/Douglas-Souza40/fctech-rate-limiter/pkg/middleware"
)

func main() {
    // load env from .env if present (best-effort)
    _ = loadDotEnv()

    redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
    redisPass := os.Getenv("REDIS_PASSWORD")
    redisDB := getEnvAsInt("REDIS_DB", 0)

    store := storage.NewRedisStorage(redisAddr, redisPass, redisDB)
    l := limiter.NewLimiter(store)

    mm := middleware.NewLimiterMiddleware(l)

    mux := http.NewServeMux()
    mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
        _, _ = w.Write([]byte("pong"))
    })

    handler := mm.Handler(mux)

    addr := getEnv("SERVER_ADDR", "0.0.0.0:8080")
    fmt.Printf("starting server on %s\n", addr)
    log.Fatal(http.ListenAndServe(addr, handler))
}

// minimal dotenv loader (only KEY=VALUE lines)
func loadDotEnv() error {
    f, err := os.Open(".env")
    if err != nil {
        return err
    }
    defer f.Close()
    var line string
    buf := make([]byte, 1024)
    for {
        n, err := f.Read(buf)
        if n > 0 {
            line += string(buf[:n])
        }
        if err != nil {
            break
        }
    }
    for _, l := range splitLines(line) {
        l = trimSpace(l)
        if l == "" || l[0] == '#' {
            continue
        }
        kv := splitFirst(l, "=")
        if kv[0] != "" {
            _ = os.Setenv(kv[0], kv[1])
        }
    }
    return nil
}

// helpers (simplified to avoid extra deps)
func splitLines(s string) []string {
    var out []string
    cur := ""
    for _, r := range s {
        if r == '\n' || r == '\r' {
            if cur != "" {
                out = append(out, cur)
                cur = ""
            }
            continue
        }
        cur += string(r)
    }
    if cur != "" {
        out = append(out, cur)
    }
    return out
}

func trimSpace(s string) string {
    i := 0
    j := len(s) - 1
    for i <= j && (s[i] == ' ' || s[i] == '\t') {
        i++
    }
    for j >= i && (s[j] == ' ' || s[j] == '\t') {
        j--
    }
    if i > j {
        return ""
    }
    return s[i : j+1]
}

func splitFirst(s, sep string) [2]string {
    idx := -1
    for i := 0; i < len(s); i++ {
        if string(s[i]) == sep {
            idx = i
            break
        }
    }
    if idx == -1 {
        return [2]string{s, ""}
    }
    return [2]string{s[:idx], s[idx+1:]}
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
    var i int
    _, err := fmt.Sscanf(v, "%d", &i)
    if err != nil {
        return fallback
    }
    return i
}
