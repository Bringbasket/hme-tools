// 认证接口处理器（统一响应包，docs/04 §2）
package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gokeep/server/internal/common"
)

type Handler struct {
	svc   *Service
	authz *common.Authz
}

func NewHandler(svc *Service, authz *common.Authz) *Handler {
	return &Handler{svc: svc, authz: authz}
}

func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.GET("/captcha", h.Captcha)
	rg.POST("/login", h.Login)
	rg.POST("/logout", h.Logout)
	rg.GET("/getInfo", h.GetInfo)
	rg.GET("/routers", h.Routers)
	// 注册（邮箱 + 可选验证码，白名单免登录）
	rg.GET("/register/config", h.RegisterConfig)
	rg.POST("/register/email-code", h.SendRegisterEmailCode)
	rg.POST("/register", h.RegisterUser)
}

// GET /auth/captcha 获取图形验证码
func (h *Handler) Captcha(c *gin.Context) {
	if !h.svc.CaptchaEnabled(c.Request.Context()) {
		c.JSON(http.StatusOK, common.OK(gin.H{
			"captchaEnabled": false,
			"img":            "",
			"uuid":           "",
			"expr":           "",
		}))
		return
	}
	uuid, img, expr, err := h.svc.NewCaptcha(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.Fail(http.StatusInternalServerError, "验证码生成失败，请重试"))
		return
	}
	c.JSON(http.StatusOK, common.OK(gin.H{
		"captchaEnabled": true,
		"img":            img,
		"uuid":           uuid,
		"expr":           expr, // 表达式明文，供前端调试与无头测试
	}))
}

// POST /auth/login 登录
func (h *Handler) Login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.Fail(http.StatusBadRequest, "请求参数不完整"))
		return
	}
	token, err := h.svc.Login(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.OK(gin.H{"token": token}))
}

// POST /auth/logout 登出
func (h *Handler) Logout(c *gin.Context) {
	if ident := common.CurrentIdentity(c); ident != nil {
		h.svc.Logout(c.Request.Context(), ident)
	}
	c.JSON(http.StatusOK, common.OK(nil))
}

// GET /auth/getInfo 当前用户信息（用户/角色/权限）
func (h *Handler) GetInfo(c *gin.Context) {
	ident := common.CurrentIdentity(c)
	if ident == nil {
		c.JSON(http.StatusUnauthorized, common.Fail(http.StatusUnauthorized, "未登录或登录已过期"))
		return
	}
	user, err := h.svc.GetUserInfo(c.Request.Context(), ident.UserID)
	if err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	roles, err := h.svc.GetRoleNames(c.Request.Context(), ident.UserID)
	if err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	perms, err := h.authz.LoadPerms(c.Request.Context(), ident.UserID)
	if err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.OK(gin.H{
		"user": gin.H{
			"userId":   user.ID,
			"username": user.Username,
			"nickname": user.Nickname,
			"phone":    user.Phone,
			"email":    user.Email,
			"avatar":   "",
		},
		"roles":       roles,
		"permissions": perms.Perms,
		"isAdmin":     perms.IsAdmin,
	}))
}

// GET /auth/routers 菜单路由树
func (h *Handler) Routers(c *gin.Context) {
	ident := common.CurrentIdentity(c)
	if ident == nil {
		c.JSON(http.StatusUnauthorized, common.Fail(http.StatusUnauthorized, "未登录或登录已过期"))
		return
	}
	perms, err := h.authz.LoadPerms(c.Request.Context(), ident.UserID)
	if err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	routers, err := h.svc.GetRouters(c.Request.Context(), ident.UserID, perms.IsAdmin)
	if err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.OK(routers))
}

// GET /auth/register/config 注册开关（免登录）
func (h *Handler) RegisterConfig(c *gin.Context) {
	c.JSON(http.StatusOK, common.OK(h.svc.GetRegisterConfig(c.Request.Context())))
}

// POST /auth/register/email-code 发送注册邮箱验证码（免登录）
func (h *Handler) SendRegisterEmailCode(c *gin.Context) {
	var body struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Email == "" {
		c.JSON(http.StatusOK, common.Fail(http.StatusBadRequest, "请求参数不完整"))
		return
	}
	devCode, err := h.svc.SendRegisterEmailCode(c.Request.Context(), body.Email)
	if err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	data := gin.H{"message": "验证码已发送，请查收邮件"}
	if devCode != "" {
		// 开发环境未配置 SMTP 时回传验证码（对齐验证码 expr 的无头测试约定）
		data["devCode"] = devCode
	}
	c.JSON(http.StatusOK, common.OK(data))
}

// POST /auth/register 邮箱注册（免登录，成功即登录）
func (h *Handler) RegisterUser(c *gin.Context) {
	var req RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Email == "" || req.Password == "" {
		c.JSON(http.StatusOK, common.Fail(http.StatusBadRequest, "请求参数不完整"))
		return
	}
	token, userID, username, nickname, _, err := h.svc.Register(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusOK, common.ToResponse(err))
		return
	}
	c.JSON(http.StatusOK, common.OK(gin.H{
		"token":    token,
		"userId":   userID,
		"username": username,
		"nickname": nickname,
	}))
}
