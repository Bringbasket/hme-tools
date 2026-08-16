// 分布式锁（SET NX PX；用于需要跨实例串行化的关键操作）。
package common

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/redis/go-redis/v9"
)

const releaseLockScript = `
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("del", KEYS[1])
end
return 0
`

// AcquireLock 获取分布式锁，成功时返回仅属于当前持有者的令牌。
func AcquireLock(ctx context.Context, rdb *redis.Client, key string, ttl time.Duration) (string, bool, error) {
	if rdb == nil {
		return "", false, nil
	}
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", false, err
	}
	token := hex.EncodeToString(tokenBytes)
	ok, err := rdb.SetNX(ctx, key, token, ttl).Result()
	return token, ok, err
}

// ReleaseLock 释放分布式锁（仅持有者释放）
func ReleaseLock(ctx context.Context, rdb *redis.Client, key, token string) error {
	if rdb == nil || token == "" {
		return nil
	}
	return rdb.Eval(ctx, releaseLockScript, []string{key}, token).Err()
}
