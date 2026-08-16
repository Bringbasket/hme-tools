// Package webui 内嵌前端构建产物（apps/web 的 vite build 输出到 server/webui/dist，见 docs/02）
package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// FS 返回 dist 子文件系统（根即 dist 内容）
func FS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}

// Index 返回首页 HTML；前端未构建时 ok=false（仅 API 模式）
func Index() ([]byte, bool) {
	b, err := distFS.ReadFile("dist/index.html")
	if err != nil {
		return nil, false
	}
	return b, true
}
