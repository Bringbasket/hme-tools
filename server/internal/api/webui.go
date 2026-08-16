// 内嵌前端静态托管：/ 与 /assets/* 由 Go embed 提供，非 API GET 走 SPA fallback
package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"gokeep/server/internal/common"
	"gokeep/server/webui"
)

func registerStatic(r *gin.Engine) {
	index, ok := webui.Index()
	dist, distErr := webui.FS()

	if ok && distErr == nil {
		r.GET("/", func(c *gin.Context) {
			c.Data(http.StatusOK, "text/html; charset=utf-8", index)
		})
		if assets, err := fs.Sub(dist, "assets"); err == nil {
			r.StaticFS("/assets", http.FS(assets))
		}
	}

	// NoRoute：/api 保持 JSON 404；其余 GET 回 SPA 首页（前端路由接管）
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, common.Fail(http.StatusNotFound, "接口不存在"))
			return
		}
		if ok && c.Request.Method == http.MethodGet {
			c.Data(http.StatusOK, "text/html; charset=utf-8", index)
			return
		}
		c.JSON(http.StatusNotFound, common.Fail(http.StatusNotFound, "页面不存在"))
	})
}
