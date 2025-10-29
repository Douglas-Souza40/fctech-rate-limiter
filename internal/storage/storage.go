package storage

import "time"

// Storage defines the persistence operations required by the limiter.
type Storage interface {
    // Increment increments the counter for a given key and returns the current count after increment.
    // The counter should expire after window seconds.
    Increment(key string, window time.Duration) (int64, error)

    // SetBlocked marks an identifier as blocked for the given duration.
    SetBlocked(key string, duration time.Duration) error

    // IsBlocked returns whether the identifier is currently blocked and remaining block duration.
    IsBlocked(key string) (bool, time.Duration, error)
}
