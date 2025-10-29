package middleware

import (
    "encoding/json"
    "net/http"
    "strings"

    "github.com/Douglas-Souza40/fctech-rate-limiter/internal/limiter"
)

type LimiterMiddleware struct {
    limiter *limiter.Limiter
}

func NewLimiterMiddleware(l *limiter.Limiter) *LimiterMiddleware {
    return &LimiterMiddleware{limiter: l}
}

func (m *LimiterMiddleware) Handler(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // get API key
        apiKey := r.Header.Get("API_KEY")

        // get IP (X-Forwarded-For or RemoteAddr)
        ip := clientIP(r)

        res, err := m.limiter.Allow(ip, apiKey)
        if err != nil {
            http.Error(w, "internal error", http.StatusInternalServerError)
            return
        }
        if !res.Allowed {
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(http.StatusTooManyRequests)
            body := map[string]string{"message": "you have reached the maximum number of requests or actions allowed within a certain time frame"}
            _ = json.NewEncoder(w).Encode(body)
            return
        }
        next.ServeHTTP(w, r)
    })
}

func clientIP(r *http.Request) string {
    // check X-Forwarded-For
    xff := r.Header.Get("X-Forwarded-For")
    if xff != "" {
        // could be comma separated
        parts := strings.Split(xff, ",")
        return strings.TrimSpace(parts[0])
    }
    // fallback to remote addr (without port)
    ra := r.RemoteAddr
    // strip port if present
    if idx := strings.LastIndex(ra, ":"); idx != -1 {
        return ra[:idx]
    }
    return ra
}
