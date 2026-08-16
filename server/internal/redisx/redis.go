// Redis 客户端：连接管理与连通性探测。
package redisx

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

func New(addr, password string) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
	})
}

// Ping 1s 超时探测
func Ping(ctx context.Context, client *redis.Client) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	return client.Ping(ctx).Err()
}
