package storage

import (
    "context"
    "time"

    "github.com/redis/go-redis/v9"
)

type RedisStorage struct {
    client *redis.Client
}

// NewRedisStorage creates a new RedisStorage.
func NewRedisStorage(addr, password string, db int) *RedisStorage {
    rdb := redis.NewClient(&redis.Options{
        Addr:     addr,
        Password: password,
        DB:       db,
    })
    return &RedisStorage{client: rdb}
}

var incrScript = redis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if tonumber(current) == 1 then
  redis.call("EXPIRE", KEYS[1], ARGV[1])
end
return current
`)

func (r *RedisStorage) Increment(key string, window time.Duration) (int64, error) {
    ctx := context.Background()
    seconds := int(window.Seconds())
    res, err := incrScript.Run(ctx, r.client, []string{key}, seconds).Result()
    if err != nil {
        return 0, err
    }
    switch v := res.(type) {
    case int64:
        return v, nil
    case uint64:
        return int64(v), nil
    case string:
        // sometimes Redis returns string
        return 0, nil
    default:
        return 0, nil
    }
}

func (r *RedisStorage) SetBlocked(key string, duration time.Duration) error {
    ctx := context.Background()
    bkey := "blocked:" + key
    return r.client.Set(ctx, bkey, "1", duration).Err()
}

func (r *RedisStorage) IsBlocked(key string) (bool, time.Duration, error) {
    ctx := context.Background()
    bkey := "blocked:" + key
    ttl, err := r.client.TTL(ctx, bkey).Result()
    if err != nil {
        // if key doesn't exist, Redis returns -2
        return false, 0, nil
    }
    if ttl <= 0 {
        return false, 0, nil
    }
    return true, ttl, nil
}
