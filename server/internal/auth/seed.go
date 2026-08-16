// 种子数据：admin 角色/账号、菜单树、参数配置（开发环境启动时幂等执行）
package auth

import (
	"context"
	"log/slog"

	"gokeep/server/internal/common"
	"gokeep/server/internal/ent"
	"gokeep/server/internal/ent/sysconfig"
	"gokeep/server/internal/ent/sysmenu"
	"gokeep/server/internal/ent/sysrole"
	"gokeep/server/internal/ent/sysrolemenu"
	"gokeep/server/internal/ent/sysuser"
	"gokeep/server/internal/ent/sysuserrole"
)

type seedMenu struct {
	name      string
	menuType  string
	path      string
	component string
	perms     string
	icon      string
	order     int
	parent    string // 父菜单 name，空为根
}

func Seed(ctx context.Context, client *ent.Client) error {
	// 1. 角色（按 code 幂等补齐，避免半截数据时缺角色）
	var adminRole, commonRole *ent.SysRole
	adminRole, _ = client.SysRole.Query().Where(sysrole.CodeEQ("admin")).Only(ctx)
	if adminRole == nil {
		adminRole, _ = client.SysRole.Create().SetName("超级管理员").SetCode("admin").SetIsAdmin(true).SetSort(1).Save(ctx)
	}
	commonRole, _ = client.SysRole.Query().Where(sysrole.CodeEQ("common")).Only(ctx)
	if commonRole == nil {
		commonRole, _ = client.SysRole.Create().SetName("普通用户").SetCode("common").SetSort(2).Save(ctx)
	}
	slog.Info("seed: 角色已就绪", "admin", adminRole != nil, "common", commonRole != nil)

	// 2. 用户（按 username 幂等补齐）
	var adminUser *ent.SysUser
	adminUser, _ = client.SysUser.Query().Where(sysuser.UsernameEQ("admin")).Only(ctx)
	if adminUser == nil {
		hash, err := common.HashPassword("admin123")
		if err != nil {
			return err
		}
		adminUser, err = client.SysUser.Create().SetUsername("admin").SetPassword(hash).SetNickname("管理员").Save(ctx)
		if err != nil {
			return err
		}
		slog.Info("seed: admin 账号已创建（admin / admin123，请尽快修改）")
	}

	// 3. 关联 admin → 超级管理员
	if adminUser != nil && adminRole != nil {
		n, _ := client.SysUserRole.Query().
			Where(sysuserrole.UserID(adminUser.ID), sysuserrole.RoleID(adminRole.ID)).
			Count(ctx)
		if n == 0 {
			_, _ = client.SysUserRole.Create().SetUserID(adminUser.ID).SetRoleID(adminRole.ID).Save(ctx)
		}
	}

	// 4. 菜单树（幂等：按 path 补齐缺失菜单）
	upgradeSettingsMenu(ctx, client)
	seedMenus(ctx, client)

	// 5. 超级管理员绑定全部菜单
	if adminRole != nil {
		linkAllMenus(ctx, client, adminRole)
	}

	// 6. 参数配置
	_ = ensureConfig(ctx, client, "sys.account.captchaEnabled", "true", "登录验证码开关")
	_ = ensureConfig(ctx, client, "sys.account.registerUser", "true", "是否开启用户注册（总开关）")
	_ = ensureConfig(ctx, client, "sys.register.emailCodeEnabled", "false", "注册邮箱验证码开关（true=需收邮件验证码，false=免验证码直接注册）")
	_ = ensureConfig(ctx, client, "sys.mail.host", "", "SMTP 服务器地址（空=未配置邮件服务）")
	_ = ensureConfig(ctx, client, "sys.mail.port", "465", "SMTP 端口（465=SSL，587=STARTTLS）")
	_ = ensureConfig(ctx, client, "sys.mail.ssl", "true", "SMTP 是否使用 SSL 直连（true=465，false=587 STARTTLS）")
	_ = ensureConfig(ctx, client, "sys.mail.username", "", "SMTP 账号")
	_ = ensureConfig(ctx, client, "sys.mail.password", "", "SMTP 密码/授权码")
	_ = ensureConfig(ctx, client, "sys.mail.from", "", "发件人邮箱")
	// 数据备份
	_ = ensureConfig(ctx, client, "sys.backup.s3.endpoint", "", "S3 端点地址")
	_ = ensureConfig(ctx, client, "sys.backup.s3.region", "", "S3 区域（留空默认 us-east-1）")
	_ = ensureConfig(ctx, client, "sys.backup.s3.bucket", "", "S3 存储桶")
	_ = ensureConfig(ctx, client, "sys.backup.s3.prefix", "backups/", "S3 Key 前缀")
	_ = ensureConfig(ctx, client, "sys.backup.s3.accessKey", "", "S3 Access Key ID")
	_ = ensureConfig(ctx, client, "sys.backup.s3.secretKey", "", "S3 Secret Access Key")
	_ = ensureConfig(ctx, client, "sys.backup.s3.forcePathStyle", "false", "S3 强制路径风格（云厂商保持 false，仅自建 MinIO 开启）")
	_ = ensureConfig(ctx, client, "sys.backup.schedule.enabled", "false", "启用定时备份")
	_ = ensureConfig(ctx, client, "sys.backup.schedule.cron", "0 2 * * *", "定时备份 Cron 表达式")
	_ = ensureConfig(ctx, client, "sys.backup.schedule.expireDays", "0", "备份过期天数（0=永不过期）")
	_ = ensureConfig(ctx, client, "sys.backup.schedule.maxKeep", "0", "最大保留备份份数（0=不限制）")
	// 7. 普通用户（注册用户）默认绑定首页
	if commonRole != nil {
		if home, err := client.SysMenu.Query().Where(sysmenu.PathEQ("/dashboard")).Only(ctx); err == nil {
			n, _ := client.SysRoleMenu.Query().
				Where(sysrolemenu.RoleID(commonRole.ID), sysrolemenu.MenuID(home.ID)).Count(ctx)
			if n == 0 {
				_, _ = client.SysRoleMenu.Create().SetRoleID(commonRole.ID).SetMenuID(home.ID).Save(ctx)
			}
		}
	}
	return nil
}

// seedMenus 幂等补齐菜单（按 path 判断，缺哪个补哪个）
func seedMenus(ctx context.Context, client *ent.Client) {
	list := []seedMenu{
		{name: "首页", menuType: "C", path: "/dashboard", component: "dashboard/index", icon: "layout-dashboard", order: 1},
		{name: "系统管理", menuType: "M", path: "/system", icon: "settings", order: 2},
		{name: "用户管理", menuType: "C", path: "/system/user", component: "system/user/index", perms: "system:user:list", icon: "users", order: 1, parent: "系统管理"},
		{name: "角色管理", menuType: "C", path: "/system/role", component: "system/role/index", perms: "system:role:list", icon: "shield-check", order: 2, parent: "系统管理"},
		{name: "菜单管理", menuType: "C", path: "/system/menu", component: "system/menu/index", perms: "system:menu:list", icon: "list-tree", order: 3, parent: "系统管理"},
		{name: "字典管理", menuType: "C", path: "/system/dict", component: "system/dict/index", perms: "system:dict:list", icon: "book-open", order: 4, parent: "系统管理"},
		{name: "系统设置", menuType: "C", path: "/system/settings", component: "system/settings/index", perms: "system:config:list", icon: "settings-2", order: 5, parent: "系统管理"},
		{name: "操作日志", menuType: "C", path: "/system/operlog", component: "system/operlog/index", perms: "system:operlog:list", icon: "scroll-text", order: 6, parent: "系统管理"},
	}
	ids := map[string]int64{}
	// 先登记已存在菜单的 name→id（幂等基准）
	existing, _ := client.SysMenu.Query().All(ctx)
	for _, m := range existing {
		ids[m.Name] = m.ID
	}
	created := 0
	// 第一遍：创建缺失菜单（不设 parent，第二遍统一关联，避免依赖创建顺序）
	for _, m := range list {
		if _, ok := ids[m.name]; ok {
			continue // 已存在（按 name 幂等）
		}
		builder := client.SysMenu.Create().
			SetName(m.name).
			SetMenuType(m.menuType).
			SetPath(m.path).
			SetOrderNum(m.order)
		if m.component != "" {
			builder.SetComponent(m.component)
		}
		if m.perms != "" {
			builder.SetPerms(m.perms)
		}
		if m.icon != "" {
			builder.SetIcon(m.icon)
		}
		menu, err := builder.Save(ctx)
		if err != nil {
			slog.Error("seed: 菜单创建失败", "name", m.name, "err", err)
			continue
		}
		ids[m.name] = menu.ID
		created++
	}
	if created > 0 {
		slog.Info("seed: 补齐菜单", "count", created)
	}

	// 第二遍：统一修正 parent_id。父菜单被重建/恢复导致 id 漂移时（如旧 id 丢失、新 id 自增），
	// 已存在子菜单的 parent_id 会指向失效 id，成为孤立节点；这里按 name 关联回正确父级。
	fixed := 0
	for _, m := range list {
		if m.parent == "" {
			continue
		}
		pid, ok := ids[m.parent]
		if !ok {
			continue
		}
		mid, ok := ids[m.name]
		if !ok {
			continue
		}
		cur, err := client.SysMenu.Get(ctx, mid)
		if err != nil || cur.ParentID == pid {
			continue
		}
		if err := client.SysMenu.UpdateOneID(mid).SetParentID(pid).Exec(ctx); err == nil {
			fixed++
			slog.Warn("seed: 修正菜单父级", "name", m.name, "parent", m.parent, "oldParentID", cur.ParentID, "newParentID", pid)
		}
	}
	if fixed > 0 {
		slog.Info("seed: 修正菜单父级关联", "count", fixed)
	}
}

// upgradeSettingsMenu 存量库升级：参数设置 → 系统设置（改名/路径/组件，保留角色绑定）
func upgradeSettingsMenu(ctx context.Context, client *ent.Client) {
	old, err := client.SysMenu.Query().Where(sysmenu.NameEQ("参数设置")).Only(ctx)
	if err != nil {
		return
	}
	_, _ = client.SysMenu.UpdateOneID(old.ID).
		SetName("系统设置").
		SetPath("/system/settings").
		SetComponent("system/settings/index").
		Save(ctx)
}

func linkAllMenus(ctx context.Context, client *ent.Client, role *ent.SysRole) {
	menus, err := client.SysMenu.Query().All(ctx)
	if err != nil {
		return
	}
	for _, m := range menus {
		n, _ := client.SysRoleMenu.Query().Where(sysrolemenu.RoleID(role.ID), sysrolemenu.MenuID(m.ID)).Count(ctx)
		if n == 0 {
			_, _ = client.SysRoleMenu.Create().SetRoleID(role.ID).SetMenuID(m.ID).Save(ctx)
		}
	}
}

func ensureConfig(ctx context.Context, client *ent.Client, key, value, remark string) error {
	n, _ := client.SysConfig.Query().Where(sysconfig.KeyEQ(key)).Count(ctx)
	if n > 0 {
		return nil
	}
	_, err := client.SysConfig.Create().SetName(key).SetKey(key).SetValue(value).SetRemark(remark).Save(ctx)
	return err
}
