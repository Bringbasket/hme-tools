// 系统管理服务：用户、角色、菜单、字典、参数和操作日志。
package system

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"gokeep/server/internal/common"
	"gokeep/server/internal/ent"
	"gokeep/server/internal/ent/sysconfig"
	"gokeep/server/internal/ent/sysdictdata"
	"gokeep/server/internal/ent/sysdicttype"
	"gokeep/server/internal/ent/sysmenu"
	"gokeep/server/internal/ent/sysoperlog"
	"gokeep/server/internal/ent/sysrole"
	"gokeep/server/internal/ent/sysrolemenu"
	"gokeep/server/internal/ent/sysuser"
	"gokeep/server/internal/ent/sysuserrole"
)

type Service struct {
	ent *ent.Client
}

func New(client *ent.Client) *Service {
	return &Service{ent: client}
}

// ==================== 用户 ====================

type UserReq struct {
	Username string  `json:"username" binding:"required,min=3,max=30"`
	Password string  `json:"password"` // 创建必填；更新为空则不修改
	Nickname string  `json:"nickname"`
	Phone    *string `json:"phone"`
	Email    *string `json:"email"`
	Status   string  `json:"status"`
	RoleIDs  []int64 `json:"roleIds"`
	Remark   *string `json:"remark"`
}

type UserView struct {
	ID        int64    `json:"id"`
	Username  string   `json:"username"`
	Nickname  string   `json:"nickname"`
	Phone     *string  `json:"phone"`
	Email     *string  `json:"email"`
	Status    string   `json:"status"`
	RoleIDs   []int64  `json:"roleIds"`
	RoleNames []string `json:"roleNames"`
	Remark    *string  `json:"remark"`
	CreatedAt string   `json:"createdAt"`
}

func (s *Service) ListUsers(ctx context.Context, pq common.PageQuery, keyword, status string) (*common.PageResult[UserView], error) {
	q := s.ent.SysUser.Query()
	if keyword != "" {
		q = q.Where(sysuser.Or(sysuser.UsernameContains(keyword), sysuser.NicknameContains(keyword)))
	}
	if status != "" {
		q = q.Where(sysuser.StatusEQ(status))
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	users, err := q.Order(ent.Desc(sysuser.FieldCreatedAt)).Offset(pq.Offset).Limit(pq.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]UserView, 0, len(users))
	for _, u := range users {
		roleIDs, roleNames, _ := s.userRoles(ctx, u.ID)
		list = append(list, UserView{
			ID: u.ID, Username: u.Username, Nickname: u.Nickname,
			Phone: u.Phone, Email: u.Email, Status: u.Status,
			RoleIDs: roleIDs, RoleNames: roleNames, Remark: u.Remark,
			CreatedAt: u.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return &common.PageResult[UserView]{List: list, Total: int64(total), Page: pq.Page, PageSize: pq.PageSize}, nil
}

func (s *Service) CreateUser(ctx context.Context, req UserReq) (int64, error) {
	if req.Password == "" {
		return 0, common.NewBizError(http.StatusBadRequest, "密码不能为空")
	}
	if n, _ := s.ent.SysUser.Query().Where(sysuser.UsernameEQ(req.Username)).Count(ctx); n > 0 {
		return 0, common.NewBizError(http.StatusConflict, "用户名已存在")
	}
	hash, err := common.HashPassword(req.Password)
	if err != nil {
		return 0, err
	}
	tx, err := s.ent.Tx(ctx)
	if err != nil {
		return 0, err
	}
	u, err := tx.SysUser.Create().
		SetUsername(req.Username).SetPassword(hash).
		SetNickname(req.Nickname).SetStatus(req.Status).
		SetNillablePhone(req.Phone).SetNillableEmail(req.Email).SetNillableRemark(req.Remark).
		Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	if err := replaceUserRoles(ctx, tx, u.ID, req.RoleIDs); err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	return u.ID, tx.Commit()
}

func (s *Service) UpdateUser(ctx context.Context, id int64, req UserReq) error {
	tx, err := s.ent.Tx(ctx)
	if err != nil {
		return err
	}
	upd := tx.SysUser.UpdateOneID(id).
		SetUsername(req.Username).SetNickname(req.Nickname).SetStatus(req.Status).
		SetNillablePhone(req.Phone).SetNillableEmail(req.Email).SetNillableRemark(req.Remark)
	if req.Password != "" {
		hash, err := common.HashPassword(req.Password)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		upd.SetPassword(hash)
	}
	if _, err := upd.Save(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := replaceUserRoles(ctx, tx, id, req.RoleIDs); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *Service) DeleteUsers(ctx context.Context, ids []int64) error {
	if contains(ids, 1) {
		return common.NewBizError(http.StatusForbidden, "不允许删除内置管理员")
	}
	tx, err := s.ent.Tx(ctx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		_, _ = tx.SysUserRole.Delete().Where(sysuserrole.UserID(id)).Exec(ctx)
	}
	if _, err := tx.SysUser.Delete().Where(sysuser.IDIn(ids...)).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *Service) userRoles(ctx context.Context, uid int64) ([]int64, []string, error) {
	ur, err := s.ent.SysUserRole.Query().Where(sysuserrole.UserID(uid)).All(ctx)
	if err != nil {
		return nil, nil, err
	}
	ids := make([]int64, 0, len(ur))
	names := make([]string, 0, len(ur))
	for _, r := range ur {
		ids = append(ids, r.RoleID)
		role, err := s.ent.SysRole.Get(ctx, r.RoleID)
		if err == nil {
			names = append(names, role.Name)
		}
	}
	return ids, names, nil
}

func replaceUserRoles(ctx context.Context, tx *ent.Tx, uid int64, roleIDs []int64) error {
	if _, err := tx.SysUserRole.Delete().Where(sysuserrole.UserID(uid)).Exec(ctx); err != nil {
		return err
	}
	for _, rid := range roleIDs {
		if _, err := tx.SysUserRole.Create().SetUserID(uid).SetRoleID(rid).Save(ctx); err != nil {
			return err
		}
	}
	return nil
}

// ==================== 角色 ====================

type RoleReq struct {
	Name    string  `json:"name" binding:"required"`
	Code    string  `json:"code" binding:"required"`
	Sort    int     `json:"sort"`
	IsAdmin bool    `json:"isAdmin"`
	Status  string  `json:"status"`
	Remark  *string `json:"remark"`
	MenuIDs []int64 `json:"menuIds"`
}

func (s *Service) ListRoles(ctx context.Context, pq common.PageQuery, keyword string) (*common.PageResult[*ent.SysRole], error) {
	q := s.ent.SysRole.Query()
	if keyword != "" {
		q = q.Where(sysrole.Or(sysrole.NameContains(keyword), sysrole.CodeContains(keyword)))
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	list, err := q.Order(ent.Asc(sysrole.FieldSort)).Offset(pq.Offset).Limit(pq.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	return &common.PageResult[*ent.SysRole]{List: list, Total: int64(total), Page: pq.Page, PageSize: pq.PageSize}, nil
}

func (s *Service) AllRoles(ctx context.Context) ([]*ent.SysRole, error) {
	return s.ent.SysRole.Query().Where(sysrole.StatusEQ("0")).Order(ent.Asc(sysrole.FieldSort)).All(ctx)
}

func (s *Service) CreateRole(ctx context.Context, req RoleReq) (int64, error) {
	if n, _ := s.ent.SysRole.Query().Where(sysrole.Or(sysrole.NameEQ(req.Name), sysrole.CodeEQ(req.Code))).Count(ctx); n > 0 {
		return 0, common.NewBizError(http.StatusConflict, "角色名称或权限字符已存在")
	}
	r, err := s.ent.SysRole.Create().
		SetName(req.Name).SetCode(req.Code).SetSort(req.Sort).
		SetIsAdmin(req.IsAdmin).SetStatus(req.Status).SetNillableRemark(req.Remark).
		Save(ctx)
	if err != nil {
		return 0, err
	}
	if err := s.AssignRoleMenus(ctx, r.ID, req.MenuIDs); err != nil {
		return 0, err
	}
	return r.ID, nil
}

func (s *Service) UpdateRole(ctx context.Context, id int64, req RoleReq) error {
	if id == 1 && !req.IsAdmin {
		return common.NewBizError(http.StatusForbidden, "不允许取消内置管理员的超级权限")
	}
	if _, err := s.ent.SysRole.UpdateOneID(id).
		SetName(req.Name).SetCode(req.Code).SetSort(req.Sort).
		SetIsAdmin(req.IsAdmin).SetStatus(req.Status).SetNillableRemark(req.Remark).
		Save(ctx); err != nil {
		return err
	}
	return s.AssignRoleMenus(ctx, id, req.MenuIDs)
}

func (s *Service) DeleteRoles(ctx context.Context, ids []int64) error {
	if contains(ids, 1) {
		return common.NewBizError(http.StatusForbidden, "不允许删除内置管理员角色")
	}
	tx, err := s.ent.Tx(ctx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if n, _ := tx.SysUserRole.Query().Where(sysuserrole.RoleID(id)).Count(ctx); n > 0 {
			_ = tx.Rollback()
			return common.NewBizError(http.StatusConflict, fmt.Sprintf("角色(ID=%d)下仍有用户，请先移除关联", id))
		}
		_, _ = tx.SysRoleMenu.Delete().Where(sysrolemenu.RoleID(id)).Exec(ctx)
	}
	if _, err := tx.SysRole.Delete().Where(sysrole.IDIn(ids...)).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *Service) AssignRoleMenus(ctx context.Context, roleID int64, menuIDs []int64) error {
	tx, err := s.ent.Tx(ctx)
	if err != nil {
		return err
	}
	if _, err := tx.SysRoleMenu.Delete().Where(sysrolemenu.RoleID(roleID)).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	for _, mid := range menuIDs {
		if _, err := tx.SysRoleMenu.Create().SetRoleID(roleID).SetMenuID(mid).Save(ctx); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *Service) RoleMenuIDs(ctx context.Context, roleID int64) ([]int64, error) {
	rows, err := s.ent.SysRoleMenu.Query().Where(sysrolemenu.RoleID(roleID)).All(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.MenuID)
	}
	return ids, nil
}

// ==================== 菜单 ====================

type MenuReq struct {
	ParentID  int64   `json:"parentId"`
	Name      string  `json:"name" binding:"required"`
	MenuType  string  `json:"menuType"`
	Path      string  `json:"path"`
	Component *string `json:"component"`
	Perms     *string `json:"perms"`
	Icon      *string `json:"icon"`
	OrderNum  int     `json:"orderNum"`
	Visible   bool    `json:"visible"`
	Status    string  `json:"status"`
}

func (s *Service) ListMenus(ctx context.Context) ([]*ent.SysMenu, error) {
	return s.ent.SysMenu.Query().Order(ent.Asc(sysmenu.FieldParentID), ent.Asc(sysmenu.FieldOrderNum)).All(ctx)
}

func (s *Service) CreateMenu(ctx context.Context, req MenuReq) (int64, error) {
	m, err := s.ent.SysMenu.Create().
		SetParentID(req.ParentID).SetName(req.Name).SetMenuType(req.MenuType).
		SetPath(req.Path).SetOrderNum(req.OrderNum).SetVisible(req.Visible).SetStatus(req.Status).
		SetNillableComponent(req.Component).SetNillablePerms(req.Perms).SetNillableIcon(req.Icon).
		Save(ctx)
	if err != nil {
		return 0, err
	}
	return m.ID, nil
}

func (s *Service) UpdateMenu(ctx context.Context, id int64, req MenuReq) error {
	_, err := s.ent.SysMenu.UpdateOneID(id).
		SetParentID(req.ParentID).SetName(req.Name).SetMenuType(req.MenuType).
		SetPath(req.Path).SetOrderNum(req.OrderNum).SetVisible(req.Visible).SetStatus(req.Status).
		SetNillableComponent(req.Component).SetNillablePerms(req.Perms).SetNillableIcon(req.Icon).
		Save(ctx)
	return err
}

// DeleteMenus 删除菜单及其子树与角色关联
func (s *Service) DeleteMenus(ctx context.Context, id int64) error {
	tx, err := s.ent.Tx(ctx)
	if err != nil {
		return err
	}
	// 收集子树
	all, err := tx.SysMenu.Query().All(ctx)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	ids := subtreeIDs(all, id)
	_, _ = tx.SysRoleMenu.Delete().Where(sysrolemenu.MenuIDIn(ids...)).Exec(ctx)
	if _, err := tx.SysMenu.Delete().Where(sysmenu.IDIn(ids...)).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func subtreeIDs(menus []*ent.SysMenu, root int64) []int64 {
	ids := []int64{root}
	for changed := true; changed; {
		changed = false
		for _, m := range menus {
			if contains(ids, m.ParentID) && !contains(ids, m.ID) {
				ids = append(ids, m.ID)
				changed = true
			}
		}
	}
	return ids
}

// ==================== 字典 ====================

type DictTypeReq struct {
	Name   string  `json:"name" binding:"required"`
	Type   string  `json:"type" binding:"required"`
	Status string  `json:"status"`
	Remark *string `json:"remark"`
}

type DictDataReq struct {
	Sort     int     `json:"sort"`
	Label    string  `json:"label" binding:"required"`
	Value    string  `json:"value" binding:"required"`
	DictType string  `json:"dictType" binding:"required"`
	Status   string  `json:"status"`
	Remark   *string `json:"remark"`
}

func (s *Service) ListDictTypes(ctx context.Context, pq common.PageQuery, keyword string) (*common.PageResult[*ent.SysDictType], error) {
	q := s.ent.SysDictType.Query()
	if keyword != "" {
		q = q.Where(sysdicttype.Or(sysdicttype.NameContains(keyword), sysdicttype.TypeContains(keyword)))
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	list, err := q.Order(ent.Asc(sysdicttype.FieldType)).Offset(pq.Offset).Limit(pq.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	return &common.PageResult[*ent.SysDictType]{List: list, Total: int64(total), Page: pq.Page, PageSize: pq.PageSize}, nil
}

func (s *Service) CreateDictType(ctx context.Context, req DictTypeReq) (int64, error) {
	if n, _ := s.ent.SysDictType.Query().Where(sysdicttype.TypeEQ(req.Type)).Count(ctx); n > 0 {
		return 0, common.NewBizError(http.StatusConflict, "字典类型已存在")
	}
	d, err := s.ent.SysDictType.Create().SetName(req.Name).SetType(req.Type).SetStatus(req.Status).SetNillableRemark(req.Remark).Save(ctx)
	if err != nil {
		return 0, err
	}
	return d.ID, nil
}

func (s *Service) UpdateDictType(ctx context.Context, id int64, req DictTypeReq) error {
	_, err := s.ent.SysDictType.UpdateOneID(id).SetName(req.Name).SetType(req.Type).SetStatus(req.Status).SetNillableRemark(req.Remark).Save(ctx)
	return err
}

func (s *Service) DeleteDictTypes(ctx context.Context, ids []int64) error {
	_, err := s.ent.SysDictType.Delete().Where(sysdicttype.IDIn(ids...)).Exec(ctx)
	return err
}

func (s *Service) ListDictData(ctx context.Context, pq common.PageQuery, dictType, keyword string) (*common.PageResult[*ent.SysDictData], error) {
	q := s.ent.SysDictData.Query()
	if dictType != "" {
		q = q.Where(sysdictdata.DictTypeEQ(dictType))
	}
	if keyword != "" {
		q = q.Where(sysdictdata.Or(sysdictdata.LabelContains(keyword), sysdictdata.ValueContains(keyword)))
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	list, err := q.Order(ent.Asc(sysdictdata.FieldSort)).Offset(pq.Offset).Limit(pq.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	return &common.PageResult[*ent.SysDictData]{List: list, Total: int64(total), Page: pq.Page, PageSize: pq.PageSize}, nil
}

// GetDictOptions 字典下拉数据（前端 useDict 数据源）
func (s *Service) GetDictOptions(ctx context.Context, dictType string) ([]gin.H, error) {
	list, err := s.ent.SysDictData.Query().
		Where(sysdictdata.DictTypeEQ(dictType), sysdictdata.StatusEQ("0")).
		Order(ent.Asc(sysdictdata.FieldSort)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]gin.H, 0, len(list))
	for _, d := range list {
		out = append(out, gin.H{"label": d.Label, "value": d.Value})
	}
	return out, nil
}

func (s *Service) CreateDictData(ctx context.Context, req DictDataReq) (int64, error) {
	d, err := s.ent.SysDictData.Create().
		SetSort(req.Sort).SetLabel(req.Label).SetValue(req.Value).
		SetDictType(req.DictType).SetStatus(req.Status).SetNillableRemark(req.Remark).
		Save(ctx)
	if err != nil {
		return 0, err
	}
	return d.ID, nil
}

func (s *Service) UpdateDictData(ctx context.Context, id int64, req DictDataReq) error {
	_, err := s.ent.SysDictData.UpdateOneID(id).
		SetSort(req.Sort).SetLabel(req.Label).SetValue(req.Value).
		SetDictType(req.DictType).SetStatus(req.Status).SetNillableRemark(req.Remark).
		Save(ctx)
	return err
}

func (s *Service) DeleteDictData(ctx context.Context, ids []int64) error {
	_, err := s.ent.SysDictData.Delete().Where(sysdictdata.IDIn(ids...)).Exec(ctx)
	return err
}

// ==================== 参数配置 ====================

type ConfigReq struct {
	Name   string  `json:"name" binding:"required"`
	Key    string  `json:"key" binding:"required"`
	Value  string  `json:"value" binding:"required"`
	Remark *string `json:"remark"`
}

func (s *Service) ListConfigs(ctx context.Context, pq common.PageQuery, keyword string) (*common.PageResult[*ent.SysConfig], error) {
	q := s.ent.SysConfig.Query()
	if keyword != "" {
		q = q.Where(sysconfig.Or(sysconfig.NameContains(keyword), sysconfig.KeyContains(keyword)))
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	list, err := q.Order(ent.Asc(sysconfig.FieldKey)).Offset(pq.Offset).Limit(pq.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	return &common.PageResult[*ent.SysConfig]{List: list, Total: int64(total), Page: pq.Page, PageSize: pq.PageSize}, nil
}

func (s *Service) CreateConfig(ctx context.Context, req ConfigReq) (int64, error) {
	if n, _ := s.ent.SysConfig.Query().Where(sysconfig.KeyEQ(req.Key)).Count(ctx); n > 0 {
		return 0, common.NewBizError(http.StatusConflict, "参数键名已存在")
	}
	cfg, err := s.ent.SysConfig.Create().SetName(req.Name).SetKey(req.Key).SetValue(req.Value).SetNillableRemark(req.Remark).Save(ctx)
	if err != nil {
		return 0, err
	}
	return cfg.ID, nil
}

func (s *Service) UpdateConfig(ctx context.Context, id int64, value string) error {
	_, err := s.ent.SysConfig.UpdateOneID(id).SetValue(value).Save(ctx)
	return err
}

func (s *Service) DeleteConfigs(ctx context.Context, ids []int64) error {
	_, err := s.ent.SysConfig.Delete().Where(sysconfig.IDIn(ids...)).Exec(ctx)
	return err
}

// GetConfigValue 供业务模块读取参数（缺失返回空字符串）
func (s *Service) GetConfigValue(ctx context.Context, key string) string {
	cfg, err := s.ent.SysConfig.Query().Where(sysconfig.KeyEQ(key)).Only(ctx)
	if err != nil {
		return ""
	}
	return cfg.Value
}

// ==================== 操作日志 ====================

func (s *Service) ListOperLogs(ctx context.Context, pq common.PageQuery, title, operator, status string) (*common.PageResult[*ent.SysOperLog], error) {
	q := s.ent.SysOperLog.Query()
	if title != "" {
		q = q.Where(sysoperlog.TitleContains(title))
	}
	if operator != "" {
		q = q.Where(sysoperlog.OperatorNameContains(operator))
	}
	if status == "fail" {
		q = q.Where(sysoperlog.StatusCodeGTE(400))
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	list, err := q.Order(ent.Desc(sysoperlog.FieldCreatedAt)).Offset(pq.Offset).Limit(pq.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	return &common.PageResult[*ent.SysOperLog]{List: list, Total: int64(total), Page: pq.Page, PageSize: pq.PageSize}, nil
}

func contains(list []int64, v int64) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}
