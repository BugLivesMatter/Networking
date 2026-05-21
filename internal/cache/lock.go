package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// unlockScript атомарно удаляет ключ только если его значение совпадает с owner.
// Предотвращает освобождение чужой блокировки при истечении TTL.
const unlockScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
else
    return 0
end`

// DistributedLock реализует распределённую блокировку на базе Redis SET NX + Lua CAS unlock.
// При rdb == nil все операции возвращают no-op (graceful degradation без Redis).
type DistributedLock struct {
	rdb *redis.Client
}

func NewDistributedLock(rdb *redis.Client) *DistributedLock {
	return &DistributedLock{rdb: rdb}
}

// Acquire пытается захватить блокировку через SET NX EX.
// Возвращает (true, nil) если блокировка захвачена,
// (false, nil) если занята другим владельцем,
// (false, err) при ошибке Redis.
// При rdb == nil всегда возвращает (true, nil) — no-op режим.
func (l *DistributedLock) Acquire(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
	if l.rdb == nil {
		return true, nil
	}
	return l.rdb.SetNX(ctx, key, owner, ttl).Result()
}

// Release освобождает блокировку атомарно через Lua-скрипт:
// удаляет ключ только если текущее значение совпадает с owner.
// При rdb == nil — no-op.
func (l *DistributedLock) Release(ctx context.Context, key, owner string) error {
	if l.rdb == nil {
		return nil
	}
	return l.rdb.Eval(ctx, unlockScript, []string{key}, owner).Err()
}
