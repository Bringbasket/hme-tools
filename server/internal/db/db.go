// Ent 客户端与迁移（docs/07 §4 / docs/03 §8）
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"gokeep/server/internal/ent"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx 驱动注册（docs/01：Ent 使用 pgx 生态）
)

// Open 创建 Ent 客户端并 Ping 校验连通（10s 超时）。
// 注意：驱动名是 pgx，方言名必须是 postgres——entsql.OpenDB(dialect.Postgres, db) 显式声明，
// 否则 ent 迁移引擎会报 unsupported dialect "pgx"。
func Open(databaseURL string) (*ent.Client, error) {
	raw, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open pgx driver: %w", err)
	}
	raw.SetMaxOpenConns(10)
	raw.SetMaxIdleConns(5)
	raw.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := raw.PingContext(ctx); err != nil {
		_ = raw.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	return ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, raw))), nil
}

// Migrate 自动迁移：仅开发环境允许（docs/03 §8：生产禁止 auto-migrate）
func Migrate(ctx context.Context, client *ent.Client, env string) error {
	if env == "production" {
		return errors.New("生产环境禁止自动迁移，请使用 deploy/migrations 脚本")
	}
	if err := client.Schema.Create(ctx); err != nil {
		return err
	}
	return migrateMailSchema(ctx, client)
}
