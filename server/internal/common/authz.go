// 认证与授权（docs/04 §4 / docs/07 §7）：
// JWT 校验 + Redis 会话存在性 + 用户状态 + 权限缓存；白名单路径跳过认证。
package common

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"gokeep/server/internal/ent"
	"gokeep/server/internal/ent/sysmenu"
	"gokeep/server/internal/ent/sysrole"
	"gokeep/server/internal/ent/sysrolemenu"
	"gokeep/server/internal/ent/sysuserrole"
)

type Identity struct {
	UserID   int64
	Username string
	IsAdmin  bool
	JTI      string
}

type userPerms struct {
	IsAdmin bool     `json:"isAdmin"`
	Perms   []string `json:"perms"`
}

const ctxIdentity = "gokeep:identity"
const permsTTL = 10 * time.Minute

type Authz struct {
	ent    *ent.Client
	rdb    *redis.Client
	secret string
	whitelist map[string]struct{}
}

func NewAuthz(client *ent.Client, rdb *redis.Client, secret string) *Authz {
	return &Authz{
		ent:    client,
		rdb:    rdb,
		secret: secret,
		whitelist: map[string]struct{}{
			"/api/v1/auth/login":             {},
			"/api/v1/auth/captcha":           {},
			"/api/v1/auth/register":          {},
			"/api/v1/auth/register/config":   {},
			"/api/v1/auth/register/email-code": {},
		},
	}
}

func CurrentIdentity(c *gin.Context) *Identity {
	v, _ := c.Get(ctxIdentity)
	ident, _ := v.(*Identity)
	return ident
}

// Middleware 认证中间件：白名单放行；否则校验 token/会话/用户状态并注入 Identity
func (a *Authz) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := a.whitelist[c.Request.URL.Path]; ok {
			c.Next()
			return
		}
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, Fail(http.StatusUnauthorized, "未登录或登录已过期"))
			return
		}
		claims, err := ParseToken(a.secret, strings.TrimPrefix(auth, "Bearer "))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, Fail(http.StatusUnauthorized, "令牌无效或已过期"))
			return
		}
		ctx := c.Request.Context()
		uidStr, err := a.rdb.Get(ctx, KeySessionPrefix+claims.JTI).Result()
		if errors.Is(err, redis.Nil) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, Fail(http.StatusUnauthorized, "会话已失效，请重新登录"))
			return
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, Fail(http.StatusUnauthorized, "会话校验失败"))
			return
		}
		uid, err := strconv.ParseInt(uidStr, 10, 64)
		if err != nil || uid != claims.UserID {
			c.AbortWithStatusJSON(http.StatusUnauthorized, Fail(http.StatusUnauthorized, "会话异常，请重新登录"))
			return
		}
		user, err := a.ent.SysUser.Get(ctx, uid)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, Fail(http.StatusUnauthorized, "用户不存在"))
			return
		}
		if user.Status != "0" {
			c.AbortWithStatusJSON(http.StatusForbidden, Fail(http.StatusForbidden, "账号已停用"))
			return
		}
		// 加载权限（缓存命中极快），设置 IsAdmin 供 Require 直接放行
		perms, _ := a.loadPerms(ctx, uid)
		ident := &Identity{UserID: uid, Username: user.Username, JTI: claims.JTI}
		if perms != nil {
			ident.IsAdmin = perms.IsAdmin
		}
		c.Set(ctxIdentity, ident)
		c.Next()
	}
}

// Require 权限中间件：IsAdmin 放行，否则校验权限字符串（如 system:user:list）
func (a *Authz) Require(perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ident := CurrentIdentity(c)
		if ident == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, Fail(http.StatusUnauthorized, "未登录或登录已过期"))
			return
		}
		if ident.IsAdmin {
			c.Next()
			return
		}
		perms, err := a.loadPerms(c.Request.Context(), ident.UserID)
		if err != nil || !contains(perms.Perms, perm) {
			c.AbortWithStatusJSON(http.StatusForbidden, Fail(http.StatusForbidden, "没有操作权限"))
			return
		}
		c.Next()
	}
}

// IsAdmin 返回当前用户是否超级管理员（供业务分支使用，如菜单树全量下发）
func (a *Authz) IsAdmin(c *gin.Context) bool {
	if ident := CurrentIdentity(c); ident != nil {
		return ident.IsAdmin
	}
	return false
}

// LoadPerms 计算并缓存用户权限（10min）；供 Authz 与业务层复用
func (a *Authz) LoadPerms(ctx context.Context, uid int64) (*userPerms, error) {
	return a.loadPerms(ctx, uid)
}

// InvalidatePerms 角色/菜单/用户角色变更后清理全部权限缓存（开发规模直接 SCAN）
func (a *Authz) InvalidatePerms(ctx context.Context) {
	iter := a.rdb.Scan(ctx, 0, KeyPermsPrefix+"*", 200).Iterator()
	for iter.Next(ctx) {
		_ = a.rdb.Del(ctx, iter.Val()).Err()
	}
}

func (a *Authz) loadPerms(ctx context.Context, uid int64) (*userPerms, error) {
	cacheKey := KeyPermsPrefix + strconv.FormatInt(uid, 10)
	if raw, err := a.rdb.Get(ctx, cacheKey).Result(); err == nil {
		var p userPerms
		if json.Unmarshal([]byte(raw), &p) == nil {
			return &p, nil
		}
	}

	// 用户 → 角色
	ur, err := a.ent.SysUserRole.Query().Where(sysuserrole.UserID(uid)).All(ctx)
	if err != nil {
		return nil, err
	}
	roleIDs := make([]int64, 0, len(ur))
	for _, r := range ur {
		roleIDs = append(roleIDs, r.RoleID)
	}
	p := &userPerms{Perms: []string{}}
	if len(roleIDs) > 0 {
		roles, err := a.ent.SysRole.Query().Where(sysrole.IDIn(roleIDs...), sysrole.StatusEQ("0")).All(ctx)
		if err != nil {
			return nil, err
		}
		activeRoleIDs := make([]int64, 0, len(roles))
		for _, r := range roles {
			activeRoleIDs = append(activeRoleIDs, r.ID)
			if r.IsAdmin {
				p.IsAdmin = true
			}
		}
		// 角色 → 菜单权限
		rm, err := a.ent.SysRoleMenu.Query().Where(sysrolemenu.RoleIDIn(activeRoleIDs...)).All(ctx)
		if err != nil {
			return nil, err
		}
		menuIDs := make([]int64, 0, len(rm))
		for _, m := range rm {
			menuIDs = append(menuIDs, m.MenuID)
		}
		if len(menuIDs) > 0 {
			menus, err := a.ent.SysMenu.Query().
				Where(sysmenu.IDIn(menuIDs...), sysmenu.StatusEQ("0"), sysmenu.PermsNotNil()).
				All(ctx)
			if err != nil {
				return nil, err
			}
			for _, m := range menus {
				if m.Perms != nil && *m.Perms != "" {
					p.Perms = append(p.Perms, *m.Perms)
				}
			}
		}
	}
	if raw, err := json.Marshal(p); err == nil {
		_ = a.rdb.Set(ctx, cacheKey, raw, permsTTL).Err()
	}
	return p, nil
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}
