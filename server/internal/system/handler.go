// 系统管理接口处理器（统一响应包 + 权限 + 权限缓存失效）
package system

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"gokeep/server/internal/backup"
	"gokeep/server/internal/common"
)

type Handler struct {
	svc    *Service
	backup *backup.Service
	authz  *common.Authz
}

func NewHandler(svc *Service, backupSvc *backup.Service, authz *common.Authz) *Handler {
	return &Handler{svc: svc, backup: backupSvc, authz: authz}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	// 用户
	rg.GET("/users", h.authz.Require("system:user:list"), h.ListUsers)
	rg.POST("/users", h.authz.Require("system:user:add"), h.CreateUser)
	rg.PUT("/users/:id", h.authz.Require("system:user:edit"), h.UpdateUser)
	rg.DELETE("/users", h.authz.Require("system:user:remove"), h.DeleteUsers)
	// 角色
	rg.GET("/roles", h.authz.Require("system:role:list"), h.ListRoles)
	rg.GET("/roles/options", h.RoleOptions)
	rg.POST("/roles", h.authz.Require("system:role:add"), h.CreateRole)
	rg.PUT("/roles/:id", h.authz.Require("system:role:edit"), h.UpdateRole)
	rg.DELETE("/roles", h.authz.Require("system:role:remove"), h.DeleteRoles)
	// 菜单
	rg.GET("/menus", h.authz.Require("system:menu:list"), h.ListMenus)
	rg.POST("/menus", h.authz.Require("system:menu:add"), h.CreateMenu)
	rg.PUT("/menus/:id", h.authz.Require("system:menu:edit"), h.UpdateMenu)
	rg.DELETE("/menus/:id", h.authz.Require("system:menu:remove"), h.DeleteMenu)
	// 字典
	rg.GET("/dict/types", h.authz.Require("system:dict:list"), h.ListDictTypes)
	rg.POST("/dict/types", h.authz.Require("system:dict:add"), h.CreateDictType)
	rg.PUT("/dict/types/:id", h.authz.Require("system:dict:edit"), h.UpdateDictType)
	rg.DELETE("/dict/types", h.authz.Require("system:dict:remove"), h.DeleteDictTypes)
	rg.GET("/dict/data", h.authz.Require("system:dict:list"), h.ListDictData)
	rg.GET("/dict/data/options", h.DictOptions)
	rg.POST("/dict/data", h.authz.Require("system:dict:add"), h.CreateDictData)
	rg.PUT("/dict/data/:id", h.authz.Require("system:dict:edit"), h.UpdateDictData)
	rg.DELETE("/dict/data", h.authz.Require("system:dict:remove"), h.DeleteDictData)
	// 参数
	rg.GET("/configs", h.authz.Require("system:config:list"), h.ListConfigs)
	rg.POST("/configs", h.authz.Require("system:config:add"), h.CreateConfig)
	rg.PUT("/configs/:id", h.authz.Require("system:config:edit"), h.UpdateConfig)
	rg.DELETE("/configs", h.authz.Require("system:config:remove"), h.DeleteConfigs)
	// 系统设置（分组化配置：功能开关 / 邮件设置 / 数据备份）
	rg.GET("/settings", h.authz.Require("system:config:list"), h.GetSettings)
	rg.PUT("/settings", h.authz.Require("system:config:edit"), h.SaveSettings)
	rg.POST("/settings/test-mail", h.authz.Require("system:config:edit"), h.TestMail)
	// 数据备份
	rg.GET("/backup/records", h.authz.Require("system:config:list"), h.ListBackupRecords)
	rg.POST("/backup/records", h.authz.Require("system:config:edit"), h.CreateBackup)
	rg.GET("/backup/records/:id/download", h.authz.Require("system:config:list"), h.DownloadBackup)
	rg.POST("/backup/records/:id/restore", h.authz.Require("system:config:edit"), h.RestoreBackup)
	rg.DELETE("/backup/records/:id", h.authz.Require("system:config:edit"), h.DeleteBackup)
	rg.POST("/backup/test-s3", h.authz.Require("system:config:edit"), h.TestS3)
	// 操作日志
	rg.GET("/operlogs", h.authz.Require("system:operlog:list"), h.ListOperLogs)
}

func page(c *gin.Context) common.PageQuery {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	return common.ParsePage(page, size)
}

func idOf(c *gin.Context) int64 {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	return id
}

func idsOf(c *gin.Context, key string) []int64 {
	raw := c.Query(key)
	if raw == "" {
		var body map[string][]int64
		_ = c.ShouldBindJSON(&body)
		return body[key]
	}
	return parseIDs(raw)
}

func parseIDs(raw string) []int64 {
	var out []int64
	start := -1
	for i := 0; i <= len(raw); i++ {
		if i < len(raw) && raw[i] >= '0' && raw[i] <= '9' {
			if start < 0 {
				start = i
			}
			continue
		}
		if start >= 0 {
			if v, err := strconv.ParseInt(raw[start:i], 10, 64); err == nil {
				out = append(out, v)
			}
			start = -1
		}
	}
	return out
}

// ==================== 用户 ====================

func (h *Handler) ListUsers(c *gin.Context) {
	users, err := h.svc.ListUsers(c.Request.Context(), page(c), c.Query("keyword"), c.Query("status"))
	if err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.OK(users))
}

func (h *Handler) CreateUser(c *gin.Context) {
	var req UserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, common.Fail(http.StatusBadRequest, "请求参数不完整"))
		return
	}
	id, err := h.svc.CreateUser(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	h.authz.InvalidatePerms(c.Request.Context())
	c.JSON(http.StatusOK, common.OK(gin.H{"id": id}))
}

func (h *Handler) UpdateUser(c *gin.Context) {
	var req UserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, common.Fail(http.StatusBadRequest, "请求参数不完整"))
		return
	}
	if err := h.svc.UpdateUser(c.Request.Context(), idOf(c), req); err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	h.authz.InvalidatePerms(c.Request.Context())
	c.JSON(http.StatusOK, common.OK(nil))
}

func (h *Handler) DeleteUsers(c *gin.Context) {
	if err := h.svc.DeleteUsers(c.Request.Context(), idsOf(c, "ids")); err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	h.authz.InvalidatePerms(c.Request.Context())
	c.JSON(http.StatusOK, common.OK(nil))
}

// ==================== 角色 ====================

func (h *Handler) ListRoles(c *gin.Context) {
	roles, err := h.svc.ListRoles(c.Request.Context(), page(c), c.Query("keyword"))
	if err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.OK(roles))
}

// RoleOptions 角色下拉（用户编辑用，登录即可）
func (h *Handler) RoleOptions(c *gin.Context) {
	roles, err := h.svc.AllRoles(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	out := make([]gin.H, 0, len(roles))
	for _, r := range roles {
		out = append(out, gin.H{"id": r.ID, "name": r.Name, "code": r.Code})
	}
	c.JSON(http.StatusOK, common.OK(out))
}

func (h *Handler) CreateRole(c *gin.Context) {
	var req RoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, common.Fail(http.StatusBadRequest, "请求参数不完整"))
		return
	}
	id, err := h.svc.CreateRole(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	h.authz.InvalidatePerms(c.Request.Context())
	c.JSON(http.StatusOK, common.OK(gin.H{"id": id}))
}

func (h *Handler) UpdateRole(c *gin.Context) {
	var req RoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, common.Fail(http.StatusBadRequest, "请求参数不完整"))
		return
	}
	if err := h.svc.UpdateRole(c.Request.Context(), idOf(c), req); err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	h.authz.InvalidatePerms(c.Request.Context())
	c.JSON(http.StatusOK, common.OK(nil))
}

func (h *Handler) DeleteRoles(c *gin.Context) {
	if err := h.svc.DeleteRoles(c.Request.Context(), idsOf(c, "ids")); err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	h.authz.InvalidatePerms(c.Request.Context())
	c.JSON(http.StatusOK, common.OK(nil))
}

// ==================== 菜单 ====================

func (h *Handler) ListMenus(c *gin.Context) {
	menus, err := h.svc.ListMenus(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.OK(menus))
}

func (h *Handler) CreateMenu(c *gin.Context) {
	var req MenuReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, common.Fail(http.StatusBadRequest, "请求参数不完整"))
		return
	}
	id, err := h.svc.CreateMenu(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	h.authz.InvalidatePerms(c.Request.Context())
	c.JSON(http.StatusOK, common.OK(gin.H{"id": id}))
}

func (h *Handler) UpdateMenu(c *gin.Context) {
	var req MenuReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, common.Fail(http.StatusBadRequest, "请求参数不完整"))
		return
	}
	if err := h.svc.UpdateMenu(c.Request.Context(), idOf(c), req); err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	h.authz.InvalidatePerms(c.Request.Context())
	c.JSON(http.StatusOK, common.OK(nil))
}

func (h *Handler) DeleteMenu(c *gin.Context) {
	if err := h.svc.DeleteMenus(c.Request.Context(), idOf(c)); err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	h.authz.InvalidatePerms(c.Request.Context())
	c.JSON(http.StatusOK, common.OK(nil))
}

// ==================== 字典 ====================

func (h *Handler) ListDictTypes(c *gin.Context) {
	list, err := h.svc.ListDictTypes(c.Request.Context(), page(c), c.Query("keyword"))
	if err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.OK(list))
}

func (h *Handler) CreateDictType(c *gin.Context) {
	var req DictTypeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, common.Fail(http.StatusBadRequest, "请求参数不完整"))
		return
	}
	id, err := h.svc.CreateDictType(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.OK(gin.H{"id": id}))
}

func (h *Handler) UpdateDictType(c *gin.Context) {
	var req DictTypeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, common.Fail(http.StatusBadRequest, "请求参数不完整"))
		return
	}
	if err := h.svc.UpdateDictType(c.Request.Context(), idOf(c), req); err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.OK(nil))
}

func (h *Handler) DeleteDictTypes(c *gin.Context) {
	if err := h.svc.DeleteDictTypes(c.Request.Context(), idsOf(c, "ids")); err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.OK(nil))
}

func (h *Handler) ListDictData(c *gin.Context) {
	list, err := h.svc.ListDictData(c.Request.Context(), page(c), c.Query("dictType"), c.Query("keyword"))
	if err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.OK(list))
}

// DictOptions 字典下拉（前端 useDict 数据源，登录即可）
func (h *Handler) DictOptions(c *gin.Context) {
	opts, err := h.svc.GetDictOptions(c.Request.Context(), c.Query("dictType"))
	if err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.OK(opts))
}

func (h *Handler) CreateDictData(c *gin.Context) {
	var req DictDataReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, common.Fail(http.StatusBadRequest, "请求参数不完整"))
		return
	}
	id, err := h.svc.CreateDictData(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.OK(gin.H{"id": id}))
}

func (h *Handler) UpdateDictData(c *gin.Context) {
	var req DictDataReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, common.Fail(http.StatusBadRequest, "请求参数不完整"))
		return
	}
	if err := h.svc.UpdateDictData(c.Request.Context(), idOf(c), req); err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.OK(nil))
}

func (h *Handler) DeleteDictData(c *gin.Context) {
	if err := h.svc.DeleteDictData(c.Request.Context(), idsOf(c, "ids")); err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.OK(nil))
}

// ==================== 参数 ====================

// GetSettings 分组设置树
func (h *Handler) GetSettings(c *gin.Context) {
	tabs, err := h.svc.GetSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.OK(tabs))
}

// SaveSettings 批量保存设置 {values: {key: value}}
func (h *Handler) SaveSettings(c *gin.Context) {
	var body struct {
		Values map[string]string `json:"values"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.Values) == 0 {
		c.JSON(http.StatusOK, common.Fail(http.StatusBadRequest, "请求参数不完整"))
		return
	}
	if err := h.svc.SaveSettings(c.Request.Context(), body.Values); err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.OK(nil))
}

// TestMail 发送测试邮件 {to}
func (h *Handler) TestMail(c *gin.Context) {
	var body struct {
		To string `json:"to"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.To == "" {
		c.JSON(http.StatusOK, common.Fail(http.StatusBadRequest, "请求参数不完整"))
		return
	}
	if err := h.svc.SendTestMail(c.Request.Context(), body.To); err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.OK(gin.H{"message": "测试邮件已发送"}))
}

func (h *Handler) ListConfigs(c *gin.Context) {
	list, err := h.svc.ListConfigs(c.Request.Context(), page(c), c.Query("keyword"))
	if err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.OK(list))
}

func (h *Handler) CreateConfig(c *gin.Context) {
	var req ConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, common.Fail(http.StatusBadRequest, "请求参数不完整"))
		return
	}
	id, err := h.svc.CreateConfig(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.OK(gin.H{"id": id}))
}

func (h *Handler) UpdateConfig(c *gin.Context) {
	var req struct {
		Value string `json:"value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, common.Fail(http.StatusBadRequest, "请求参数不完整"))
		return
	}
	if err := h.svc.UpdateConfig(c.Request.Context(), idOf(c), req.Value); err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.OK(nil))
}

func (h *Handler) DeleteConfigs(c *gin.Context) {
	if err := h.svc.DeleteConfigs(c.Request.Context(), idsOf(c, "ids")); err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.OK(nil))
}

// ==================== 数据备份 ====================

// ListBackupRecords 备份记录分页
func (h *Handler) ListBackupRecords(c *gin.Context) {
	result, err := h.backup.ListRecords(c.Request.Context(), page(c))
	if err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.OK(result))
}

// CreateBackup 手动创建备份：{expireDays}
func (h *Handler) CreateBackup(c *gin.Context) {
	var body struct {
		ExpireDays int `json:"expireDays"`
	}
	_ = c.ShouldBindJSON(&body)
	rec, err := h.backup.CreateBackup(c.Request.Context(), "manual", body.ExpireDays)
	if err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.OK(gin.H{"id": rec.ID, "status": rec.Status}))
}

// DownloadBackup 下载备份文件（流式返回 gzip）
func (h *Handler) DownloadBackup(c *gin.Context) {
	obj, fileName, err := h.backup.DownloadRecord(c.Request.Context(), idOf(c))
	if err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	defer obj.Close()
	c.Header("Content-Disposition", `attachment; filename="`+fileName+`"`)
	c.Header("Content-Type", "application/gzip")
	c.DataFromReader(http.StatusOK, -1, "application/octet-stream", obj, nil)
}

// RestoreBackup 恢复备份（清空全库后重放备份 SQL，危险操作）
func (h *Handler) RestoreBackup(c *gin.Context) {
	if err := h.backup.RestoreRecord(c.Request.Context(), idOf(c)); err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.OK(nil))
}

// DeleteBackup 删除备份（S3 对象 + 记录）
func (h *Handler) DeleteBackup(c *gin.Context) {
	if err := h.backup.DeleteRecord(c.Request.Context(), idOf(c)); err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.OK(nil))
}

// TestS3 测试 S3 连接（使用当前保存的配置）
func (h *Handler) TestS3(c *gin.Context) {
	cfg := backup.LoadS3Config(c.Request.Context(), h.backup.Ent())
	if err := cfg.TestConnection(c.Request.Context()); err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.OK(gin.H{"message": "S3 连接成功"}))
}

// ==================== 操作日志 ====================

func (h *Handler) ListOperLogs(c *gin.Context) {
	list, err := h.svc.ListOperLogs(c.Request.Context(), page(c), c.Query("title"), c.Query("operator"), c.Query("status"))
	if err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.OK(list))
}
