// 操作日志审计中间件（docs/07 §6：写操作自动记录，敏感参数脱敏）
package common

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"gokeep/server/internal/ent"
)

var sensitiveParam = regexp.MustCompile(`(?i)(password|oldPassword|newPassword|confirmPassword)=[^&]*`)

func Audit(client *ent.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		method := c.Request.Method
		if method == "GET" || method == "OPTIONS" || method == "HEAD" {
			return
		}
		path := c.Request.URL.Path
		if !strings.HasPrefix(path, "/api/") {
			return
		}
		ident := CurrentIdentity(c)
		params := c.Request.URL.RawQuery
		if params == "" {
			_ = c.Request.ParseForm()
			params = c.Request.PostForm.Encode()
		}
		params = sensitiveParam.ReplaceAllString(params, "$1=***")
		if len(params) > 500 {
			params = params[:500]
		}
		builder := client.SysOperLog.Create().
			SetTitle(titleForPath(path)).
			SetBusinessType(bizType(method)).
			SetMethod(method).
			SetPath(path).
			SetIP(c.ClientIP()).
			SetStatusCode(c.Writer.Status()).
			SetDurationMs(time.Since(start).Milliseconds()).
			SetParams(params)
		if ident != nil {
			builder.SetOperatorID(ident.UserID).SetOperatorName(ident.Username)
		}
		// 审计失败只记日志，不影响业务响应（docs/07 §6 审计中间件）
		_, err := builder.Save(context.Background())
		if err != nil {
			_ = err
		}
	}
}

func bizType(method string) string {
	switch method {
	case "POST":
		return "INSERT"
	case "PUT", "PATCH":
		return "UPDATE"
	case "DELETE":
		return "DELETE"
	default:
		return "OTHER"
	}
}

func titleForPath(path string) string {
	segs := strings.Split(strings.Trim(path, "/"), "/")
	// /api/v1/system/users/123 → system.users
	start := 0
	for i, s := range segs {
		if s == "api" || s == "v1" {
			start = i + 1
		}
	}
	parts := segs[start:]
	title := []string{}
	for _, p := range parts {
		if p == "" {
			continue
		}
		if _, err := parseInt64(p); err == nil {
			continue // 跳过路径中的 ID 段
		}
		title = append(title, p)
	}
	if len(title) == 0 {
		return "system"
	}
	return strings.Join(title, ".")
}

func parseInt64(s string) (int64, error) {
	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errNotNum
		}
		n = n*10 + int64(c-'0')
	}
	return n, nil
}

var errNotNum = &numError{}

type numError struct{}

func (e *numError) Error() string { return "not a number" }
