// 公共中间件（docs/07 §3：request-id → recovery → logging → cors）
package common

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
)

const ctxRequestID = "request_id"

// RequestID 生成 16 字节随机 request_id 并注入上下文与响应头
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := newRequestID()
		c.Set(ctxRequestID, id)
		c.Header("X-Request-Id", id)
		c.Next()
	}
}

// Recovery 捕获 panic，记录堆栈并返回统一 500（docs/07 §3）
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered", "err", err, "stack", string(debug.Stack()), "request_id", c.GetString(ctxRequestID))
				c.AbortWithStatusJSON(http.StatusInternalServerError, Fail(http.StatusInternalServerError, "系统繁忙，请稍后重试"))
			}
		}()
		c.Next()
	}
}

// AccessLog 输出结构化访问日志，不记录请求体或认证凭据。
func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		slog.Info("access",
			"request_id", c.GetString(ctxRequestID),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"ip", c.ClientIP(),
		)
	}
}

// CORS 白名单（docs/07 §3；生产由 Nginx 收敛同源请求）
func CORS(allowed []string) gin.HandlerFunc {
	set := make(map[string]struct{}, len(allowed))
	for _, o := range allowed {
		set[o] = struct{}{}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}
		if _, ok := set[origin]; ok {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key, X-Request-Id")
			c.Header("Access-Control-Max-Age", "3600")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func newRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return time.Now().Format("20060102150405")
	}
	return hex.EncodeToString(b)
}
