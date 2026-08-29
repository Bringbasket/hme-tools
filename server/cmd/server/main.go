// GoKeep 网关入口（docs/02-总体架构与目录规范 §1：单网关单二进制）
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"gokeep/server/internal/api"
	"gokeep/server/internal/auth"
	"gokeep/server/internal/common"
	"gokeep/server/internal/config"
	"gokeep/server/internal/db"
	"gokeep/server/internal/ent"
	"gokeep/server/internal/redisx"
)

var (
	buildVersion  string
	buildRevision string
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		slog.Error("gokeep server stopped with error", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg := config.Load()
	if buildVersion != "" {
		cfg.Version = buildVersion
	}
	if buildRevision != "" {
		cfg.Revision = buildRevision
	}
	if cfg.AppEnv == "production" && cfg.JWTSecret == "" {
		return errors.New("JWT_SECRET 必须配置（生产环境）")
	}
	if cfg.AppEnv == "production" && cfg.DatabaseURL == "" {
		return errors.New("DATABASE_URL 必须配置（生产环境）")
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	// 中间件顺序（docs/02 §4）：request-id → recovery → logging → cors →（audit）
	r.Use(common.RequestID(), common.Recovery(), common.AccessLog(), common.CORS(cfg.CORSAllowedOrigins))

	// 存活探针：进程存活即 200
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, common.OK("ok"))
	})

	// 数据库（未配置 DATABASE_URL 时网关仍可启动，/readyz 报告未就绪）
	var entClient *ent.Client
	if cfg.DatabaseURL != "" {
		client, err := db.Open(cfg.DatabaseURL)
		if err != nil {
			return fmt.Errorf("database open failed: %w", err)
		}
		if cfg.AppEnv == "production" {
			slog.Info("database auto-migrate skipped in production")
		} else {
			if err := db.Migrate(ctx, client, cfg.AppEnv); err != nil {
				_ = client.Close()
				return fmt.Errorf("database migrate failed: %w", err)
			}
			slog.Info("database schema migrated")
		}
		entClient = client
		defer client.Close()

		if cfg.AppEnv == "development" {
			if err := auth.Seed(ctx, client); err != nil {
				slog.Error("seed failed", "err", err)
			} else {
				slog.Info("seed data ensured")
			}
		}
	}

	// 审计中间件（依赖 Ent；未连库时跳过）
	if entClient != nil {
		r.Use(common.Audit(entClient))
	}

	// Redis（会话/验证码/权限缓存；未配置时认证不可用）
	var rdb *redis.Client
	if cfg.RedisAddr != "" {
		rdb = redisx.New(cfg.RedisAddr, cfg.RedisPassword)
		if err := redisx.Ping(ctx, rdb); err != nil {
			_ = rdb.Close()
			rdb = nil
			if cfg.AppEnv == "production" {
				return fmt.Errorf("redis ping failed: %w", err)
			}
			slog.Warn("redis unavailable; protected routes disabled", "err", err)
		} else {
			defer rdb.Close()
		}
	}
	if cfg.AppEnv == "production" && entClient == nil {
		return errors.New("PostgreSQL 未就绪（生产环境）")
	}
	if cfg.AppEnv == "production" && rdb == nil {
		return errors.New("Redis 未就绪（生产环境）")
	}

	// 就绪探针：PG / Redis 连通性（docs/09 §7）
	r.GET("/readyz", func(c *gin.Context) {
		pgOK := entClient != nil
		redisOK := rdb != nil && redisx.Ping(c.Request.Context(), rdb) == nil
		status := gin.H{"postgres": pgOK, "redis": redisOK}
		if pgOK && redisOK {
			c.JSON(http.StatusOK, common.OK(status))
			return
		}
		c.JSON(http.StatusServiceUnavailable, common.Fail(http.StatusServiceUnavailable, "依赖未就绪: "+readyHint(pgOK, redisOK)))
	})

	// 业务路由
	var runtime *api.Runtime
	if entClient != nil && rdb != nil {
		componentCtx, cancelComponents := context.WithCancel(ctx)
		registered, err := api.RegisterRoutes(componentCtx, r, entClient, rdb, cfg)
		if err != nil {
			cancelComponents()
			return fmt.Errorf("application runtime init failed: %w", err)
		}
		runtime = registered
		defer func() {
			cancelComponents()
			if err := runtime.Close(); err != nil {
				slog.Warn("application runtime close failed", "err", err)
			}
		}()
	} else {
		api.RegisterPingOnly(r)
	}

	srv := &http.Server{Addr: cfg.ServerAddr, Handler: r, ReadHeaderTimeout: 10 * time.Second}
	serveErr := make(chan error, 1)
	go func() {
		slog.Info("gokeep server listening", "addr", cfg.ServerAddr, "env", cfg.AppEnv)
		serveErr <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
	case err := <-serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("http server exited: %w", err)
		}
	}

	slog.Info("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http server shutdown: %w", err)
	}
	slog.Info("server stopped")
	return nil
}

func readyHint(pgOK, redisOK bool) string {
	switch {
	case !pgOK && !redisOK:
		return "PostgreSQL 未配置 / Redis 未连通"
	case !pgOK:
		return "PostgreSQL 未配置（DATABASE_URL）"
	default:
		return "Redis 未连通（REDIS_ADDR）"
	}
}
