// 认证服务：验证码 / 登录 / 登出 / 用户信息 / 菜单路由（docs/04 §4）
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"gokeep/server/internal/common"
	"gokeep/server/internal/ent"
	"gokeep/server/internal/ent/sysconfig"
	"gokeep/server/internal/ent/sysmenu"
	"gokeep/server/internal/ent/sysrole"
	"gokeep/server/internal/ent/sysrolemenu"
	"gokeep/server/internal/ent/sysuser"
	"gokeep/server/internal/ent/sysuserrole"
)

const (
	captchaTTL = 5 * time.Minute
)

type Service struct {
	ent        *ent.Client
	rdb        *redis.Client
	secret     string
	env        string
	sessionTTL time.Duration
}

func New(client *ent.Client, rdb *redis.Client, secret string, env string, sessionTTL time.Duration) *Service {
	return &Service{ent: client, rdb: rdb, secret: secret, env: env, sessionTTL: sessionTTL}
}

// NewCaptcha 生成数学验证码：SVG data URI + Redis 存答案（5min）
func (s *Service) NewCaptcha(ctx context.Context) (uuid, imageURI, expr string, err error) {
	capt, err := common.NewMathCaptcha()
	if err != nil {
		return "", "", "", err
	}
	uuid = randomHex(16)
	if err := s.rdb.Set(ctx, common.KeyCaptchaPrefix+uuid, capt.Answer, captchaTTL).Err(); err != nil {
		return "", "", "", err
	}
	return uuid, capt.ImageDataURI, capt.Expression, nil
}

type LoginReq struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
	CaptchaUUID string `json:"captchaUuid"`
	CaptchaCode string `json:"captchaCode"`
}

// Login 验证码 → 账号 → 密码 → JWT + Redis 会话（docs/04 §4；防撞库不暴露账号状态）
func (s *Service) Login(ctx context.Context, req LoginReq) (string, error) {
	if s.CaptchaEnabled(ctx) {
		if strings.TrimSpace(req.CaptchaUUID) == "" || strings.TrimSpace(req.CaptchaCode) == "" {
			return "", common.NewBizError(http.StatusBadRequest, "请输入验证码")
		}
		key := common.KeyCaptchaPrefix + req.CaptchaUUID
		answer, err := s.rdb.Get(ctx, key).Result()
		if errors.Is(err, redis.Nil) {
			return "", common.NewBizError(http.StatusBadRequest, "验证码已过期，请刷新后重试")
		}
		if err != nil {
			return "", err
		}
		_ = s.rdb.Del(ctx, key).Err()
		if !strings.EqualFold(answer, req.CaptchaCode) {
			return "", common.NewBizError(http.StatusBadRequest, "验证码错误")
		}
	}
	user, err := s.ent.SysUser.Query().Where(sysuser.UsernameEQ(req.Username)).Only(ctx)
	if err != nil || !common.CheckPassword(user.Password, req.Password) {
		return "", common.NewBizError(http.StatusBadRequest, "用户名或密码错误")
	}
	if user.Status != "0" {
		return "", common.NewBizError(http.StatusForbidden, "账号已停用，请联系管理员")
	}
	return s.issueSession(ctx, user.ID)
}

// issueSession 签发 JWT 并登记 Redis 会话（登录/注册共用；有效期取配置 SESSION_TTL）
func (s *Service) issueSession(ctx context.Context, userID int64) (string, error) {
	jti := randomHex(16)
	token, err := common.GenerateToken(s.secret, userID, jti, s.sessionTTL)
	if err != nil {
		return "", err
	}
	if err := s.rdb.Set(ctx, common.KeySessionPrefix+jti, strconv.FormatInt(userID, 10), s.sessionTTL).Err(); err != nil {
		return "", err
	}
	return token, nil
}

// Logout 删除会话与权限缓存（docs/04 §4）
func (s *Service) Logout(ctx context.Context, ident *common.Identity) {
	_ = s.rdb.Del(ctx, common.KeySessionPrefix+ident.JTI).Err()
	_ = s.rdb.Del(ctx, common.KeyPermsPrefix+strconv.FormatInt(ident.UserID, 10)).Err()
}

// GetUserInfo 用户基本信息
func (s *Service) GetUserInfo(ctx context.Context, uid int64) (*ent.SysUser, error) {
	return s.ent.SysUser.Get(ctx, uid)
}

// GetRoleNames 用户角色名列表
func (s *Service) GetRoleNames(ctx context.Context, uid int64) ([]string, error) {
	ur, err := s.ent.SysUserRole.Query().Where(sysuserrole.UserID(uid)).All(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(ur))
	for _, r := range ur {
		ids = append(ids, r.RoleID)
	}
	if len(ids) == 0 {
		return []string{}, nil
	}
	roles, err := s.ent.SysRole.Query().Where(sysrole.IDIn(ids...), sysrole.StatusEQ("0")).Order(ent.Asc(sysrole.FieldSort)).All(ctx)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(roles))
	for _, r := range roles {
		names = append(names, r.Name)
	}
	return names, nil
}

// MenuNode 前端动态路由节点（docs/05 §5 路由 meta 约定）
type MenuNode struct {
	ID        int64       `json:"id"`
	ParentID  int64       `json:"parentId"`
	Title     string      `json:"title"`
	Path      string      `json:"path"`
	Component string      `json:"component"`
	MenuType  string      `json:"menuType"`
	Icon      string      `json:"icon,omitempty"`
	Hidden    bool        `json:"hidden"`
	Perms     string      `json:"perms,omitempty"`
	Children  []*MenuNode `json:"children,omitempty"`
}

// GetRouters 菜单树：admin 全量；其他按角色过滤（docs/05 §5 动态路由来源）
func (s *Service) GetRouters(ctx context.Context, uid int64, isAdmin bool) ([]*MenuNode, error) {
	var menus []*ent.SysMenu
	var err error
	if isAdmin {
		menus, err = s.ent.SysMenu.Query().
			Where(sysmenu.StatusEQ("0"), sysmenu.MenuTypeNEQ("F")).
			Order(ent.Asc(sysmenu.FieldParentID), ent.Asc(sysmenu.FieldOrderNum)).
			All(ctx)
	} else {
		ur, e := s.ent.SysUserRole.Query().Where(sysuserrole.UserID(uid)).All(ctx)
		if e != nil {
			return nil, e
		}
		roleIDs := make([]int64, 0, len(ur))
		for _, r := range ur {
			roleIDs = append(roleIDs, r.RoleID)
		}
		if len(roleIDs) == 0 {
			return []*MenuNode{}, nil
		}
		rm, e := s.ent.SysRoleMenu.Query().Where(sysrolemenu.RoleIDIn(roleIDs...)).All(ctx)
		if e != nil {
			return nil, e
		}
		menuIDs := make([]int64, 0, len(rm))
		for _, m := range rm {
			menuIDs = append(menuIDs, m.MenuID)
		}
		menus, err = s.ent.SysMenu.Query().
			Where(sysmenu.IDIn(menuIDs...), sysmenu.StatusEQ("0"), sysmenu.MenuTypeNEQ("F")).
			Order(ent.Asc(sysmenu.FieldParentID), ent.Asc(sysmenu.FieldOrderNum)).
			All(ctx)
	}
	if err != nil {
		return nil, err
	}
	return buildTree(menus), nil
}

func buildTree(menus []*ent.SysMenu) []*MenuNode {
	nodes := make(map[int64]*MenuNode, len(menus))
	roots := make([]*MenuNode, 0)
	order := make(map[int64]int, len(menus))
	for i, m := range menus {
		order[m.ID] = i
	}
	for _, m := range menus {
		nodes[m.ID] = &MenuNode{
			ID:        m.ID,
			ParentID:  m.ParentID,
			Title:     m.Name,
			Path:      m.Path,
			Component: "",
			MenuType:  m.MenuType,
			Icon:      "",
			Hidden:    !m.Visible,
			Perms:     "",
		}
		if m.Component != nil {
			nodes[m.ID].Component = *m.Component
		}
		if m.Icon != nil {
			nodes[m.ID].Icon = *m.Icon
		}
		if m.Perms != nil {
			nodes[m.ID].Perms = *m.Perms
		}
	}
	for _, m := range menus {
		node := nodes[m.ID]
		if m.ParentID == 0 {
			roots = append(roots, node)
			continue
		}
		if parent, ok := nodes[m.ParentID]; ok {
			parent.Children = append(parent.Children, node)
		} else {
			roots = append(roots, node)
		}
	}
	sort.Slice(roots, func(i, j int) bool { return order[roots[i].ID] < order[roots[j].ID] })
	return roots
}

// CaptchaEnabled 返回登录验证码开关；配置缺失或读取失败时默认开启。
func (s *Service) CaptchaEnabled(ctx context.Context) bool {
	cfg, err := s.ent.SysConfig.Query().Where(sysconfig.KeyEQ("sys.account.captchaEnabled")).Only(ctx)
	if err != nil {
		return true // 配置缺失时默认开启（安全优先）
	}
	return strings.EqualFold(cfg.Value, "true")
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b)
}
