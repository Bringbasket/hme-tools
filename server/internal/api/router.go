// 路由注册与依赖装配（docs/02 §4 请求处理链路）
package api

import (
	"context"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"gokeep/server/internal/auth"
	"gokeep/server/internal/backup"
	"gokeep/server/internal/common"
	"gokeep/server/internal/config"
	"gokeep/server/internal/ent"
	"gokeep/server/internal/mail"
	"gokeep/server/internal/platform/persistence"
	"gokeep/server/internal/platform/systemupdate"
	"gokeep/server/internal/system"
)

// Runtime 持有路由注册过程中创建的后台资源。
type Runtime struct {
	mail *mail.Module
}

// Close 释放路由层持有的 Redis 连接。
func (rt *Runtime) Close() error {
	if rt != nil && rt.mail != nil {
		rt.mail.Stop()
	}
	return nil
}

// RegisterRoutes 注册全部业务路由和后台调度器。
func RegisterRoutes(ctx context.Context, r *gin.Engine, client *ent.Client, rdb *redis.Client, cfg *config.Config) (*Runtime, error) {
	registerPing(r)
	registerStatic(r)

	// 认证与授权（白名单：/auth/login、/auth/captcha）
	authz := common.NewAuthz(client, rdb, cfg.JWTSecret)
	authSvc := auth.New(client, rdb, cfg.JWTSecret, cfg.AppEnv, cfg.SessionTTL)
	auth.NewHandler(authSvc, authz).Register(r.Group("/api/v1/auth", authz.Middleware()))

	// 系统管理
	sysGroup := r.Group("/api/v1/system", authz.Middleware())
	backupSvc := backup.New(client, cfg.DatabaseURL)
	system.NewHandler(system.New(client), backupSvc, authz).Register(sysGroup)
	updateSvc := systemupdate.New(filepath.Join(cfg.DataDir, "system"), cfg.Version, cfg.Revision, cfg.RepositoryURL)
	systemupdate.NewHandler(updateSvc).Register(
		sysGroup,
		authz.Require("system:config:list"),
		authz.Require("system:config:edit"),
	)

	// 邮件模块复用 GoKeep 已打开的 PostgreSQL/Redis 连接。邮件模块使用
	// net/http ServeMux，因此通过 Gin 的 WrapH 挂载到统一 API 前缀，认证
	// 和权限仍由 GoKeep 的中间件负责。
	mailPersistence := persistence.NewShared(client.SQLDB(), rdb)
	mailModule, err := mail.NewModuleWithPersistence(cfg.MailDataDir, "", "", mailPersistence)
	if err != nil {
		return nil, err
	}
	mailMux := http.NewServeMux()
	passthrough := func(next http.Handler) http.Handler { return next }
	mailModule.RegisterRoutesAt(mailMux, passthrough, "/api/v1/mail")
	mailHandler := gin.WrapH(mailMux)
	r.Any("/api/v1/mail", authz.Middleware(), authz.Require("mail:access"), mailHandler)
	r.Any("/api/v1/mail/*path", authz.Middleware(), authz.Require("mail:access"), mailHandler)
	// 分享读取接口是邮件模块明确设计的公开入口；管理接口仍受上面的
	// JWT/权限保护。仅挂载 /share，避免与管理前端的 /mail 页面冲突。
	r.Any("/share", mailHandler)
	r.Any("/share/*path", mailHandler)
	mailModule.Start()

	// 所有组件初始化成功后再启动后台调度，避免初始化失败时遗留 goroutine。
	backupSvc.StartScheduler(ctx)
	return &Runtime{mail: mailModule}, nil
}

// RegisterPingOnly 依赖未就绪时的降级路由（仅 ping/healthz/readyz 可用）
func RegisterPingOnly(r *gin.Engine) {
	registerPing(r)
	registerNoRoute(r)
}

func registerPing(r *gin.Engine) {
	// 链路验证接口：前端 /dashboard 用它证明「vite 代理 → gin → 统一响应包」
	r.GET("/api/v1/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, common.OK(gin.H{
			"message": "pong",
			"time":    time.Now().Format(time.RFC3339),
		}))
	})
}

func registerNoRoute(r *gin.Engine) {
	// 未匹配路由统一返回响应包（docs/04 §2：所有接口含失败都返回统一结构）
	r.NoRoute(func(c *gin.Context) {
		if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
			c.JSON(http.StatusNotFound, common.Fail(http.StatusNotFound, "接口不存在"))
			return
		}
		c.JSON(http.StatusNotFound, common.Fail(http.StatusNotFound, "页面不存在"))
	})
}
