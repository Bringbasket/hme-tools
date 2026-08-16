// 系统设置：Tab 分组、分区展示和统一保存。
// 底层存储仍是 sys_configs（key/value），本文件定义分组注册表作为唯一事实来源
package system

import (
	"context"
	"net/http"
	"strings"

	"gokeep/server/internal/auth"
	"gokeep/server/internal/common"
	"gokeep/server/internal/ent"
	"gokeep/server/internal/ent/sysconfig"
)

// SettingItem 单个配置项（type: switch | text | number | password）
type SettingItem struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Type        string `json:"type"`
	Value       string `json:"value"`
	Placeholder string `json:"placeholder,omitempty"`
	Hint        string `json:"hint,omitempty"`
	Default     string `json:"-"`
}

// SettingSection 设置分区（卡片）
type SettingSection struct {
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Items       []SettingItem `json:"items"`
}

// SettingTab 设置分组（Tab 页）
type SettingTab struct {
	Key      string           `json:"key"`
	Title    string           `json:"title"`
	Sections []SettingSection `json:"sections"`
}

// settingsTabs 分组注册表：新增配置项在此登记，前端自动渲染
var settingsTabs = []SettingTab{
	{
		Key:   "features",
		Title: "功能开关",
		Sections: []SettingSection{
			{
				Title:       "账号与注册",
				Description: "登录与注册相关的全局开关，保存后立即生效",
				Items: []SettingItem{
					{Key: "sys.account.captchaEnabled", Label: "登录验证码", Type: "switch", Default: "true",
						Hint: "关闭后登录不再校验图形验证码（人机验证）"},
					{Key: "sys.account.registerUser", Label: "用户注册", Type: "switch", Default: "true",
						Hint: "注册总开关；关闭后登录页显示“注册暂未开放”"},
					{Key: "sys.register.emailCodeEnabled", Label: "注册邮箱验证码", Type: "switch", Default: "false",
						Hint: "开启后注册必须填写邮件验证码；关闭则免验证码直接注册"},
				},
			},
		},
	},
	{
		Key:   "mail",
		Title: "邮件设置",
		Sections: []SettingSection{
			{
				Title:       "SMTP 服务",
				Description: "用于发送注册验证码等系统邮件；未配置主机时邮件功能不可用（开发环境注册验证码走 devCode）",
				Items: []SettingItem{
					{Key: "sys.mail.host", Label: "SMTP 主机", Type: "text", Default: "",
						Placeholder: "smtp.example.com", Hint: "留空表示未启用邮件服务"},
					{Key: "sys.mail.port", Label: "SMTP 端口", Type: "number", Default: "465",
						Placeholder: "465", Hint: "465 = SSL 直连；587 = STARTTLS"},
					{Key: "sys.mail.ssl", Label: "使用 SSL", Type: "switch", Default: "true",
						Hint: "开启走 465 SSL 直连；关闭走 587 STARTTLS"},
					{Key: "sys.mail.username", Label: "SMTP 用户名", Type: "text", Default: "",
						Placeholder: "your-email@example.com"},
					{Key: "sys.mail.password", Label: "SMTP 密码/授权码", Type: "password", Default: "",
						Placeholder: "留空表示不修改", Hint: "出于安全不回显；保存时留空则保持当前值"},
					{Key: "sys.mail.from", Label: "发件人邮箱", Type: "text", Default: "",
						Placeholder: "noreply@example.com", Hint: "建议与 SMTP 用户名一致"},
				},
			},
		},
	},
	{
		Key:   "backup",
		Title: "数据备份",
		Sections: []SettingSection{
			{
				Title:       "S3 存储配置",
				Description: "备份文件存储到 S3 兼容对象存储（支持 MinIO / Cloudflare R2 / 阿里云 OSS 等）",
				Items: []SettingItem{
					{Key: "sys.backup.s3.endpoint", Label: "端点地址", Type: "text", Default: "",
						Placeholder: "https://<account_id>.r2.cloudflarestorage.com", Hint: "MinIO 本地示例：http://127.0.0.1:9000"},
					{Key: "sys.backup.s3.region", Label: "区域", Type: "text", Default: "",
						Placeholder: "auto", Hint: "留空默认 us-east-1"},
					{Key: "sys.backup.s3.bucket", Label: "存储桶", Type: "text", Default: "",
						Placeholder: "gokeep-backup", Hint: "需先在对象存储中创建该桶"},
					{Key: "sys.backup.s3.prefix", Label: "Key 前缀", Type: "text", Default: "backups/",
						Placeholder: "backups/"},
					{Key: "sys.backup.s3.accessKey", Label: "Access Key ID", Type: "text", Default: "",
						Placeholder: "access-key"},
					{Key: "sys.backup.s3.secretKey", Label: "Secret Access Key", Type: "password", Default: "",
						Placeholder: "留空表示不修改", Hint: "出于安全不回显；保存时留空则保持当前值"},
					{Key: "sys.backup.s3.forcePathStyle", Label: "强制路径风格", Type: "switch", Default: "false",
						Hint: "Cloudflare R2 / 阿里云 OSS / 腾讯云 COS / AWS S3 请保持关闭；仅自建 MinIO 需开启"},
				},
			},
			{
				Title:       "定时备份",
				Description: "配置自动定时备份；每次触发时读取最新配置，修改后无需重启",
				Items: []SettingItem{
					{Key: "sys.backup.schedule.enabled", Label: "启用定时备份", Type: "switch", Default: "false",
						Hint: "开启后按下方 Cron 表达式自动备份"},
					{Key: "sys.backup.schedule.cron", Label: "Cron 表达式", Type: "text", Default: "0 2 * * *",
						Placeholder: "0 2 * * *", Hint: "例如 \"0 2 * * *\" 表示每天凌晨 2 点"},
					{Key: "sys.backup.schedule.expireDays", Label: "备份过期天数", Type: "number", Default: "0",
						Placeholder: "0", Hint: "备份文件超过此天数后自动删除，0 = 永不过期"},
					{Key: "sys.backup.schedule.maxKeep", Label: "最大保留份数", Type: "number", Default: "0",
						Placeholder: "0", Hint: "最多保留的备份数量，0 = 不限制"},
				},
			},
		},
	},
}

func registryItems() []SettingItem {
	out := make([]SettingItem, 0)
	for _, tab := range settingsTabs {
		for _, sec := range tab.Sections {
			out = append(out, sec.Items...)
		}
	}
	return out
}

// GetSettings 返回分组设置树（值来自 sys_configs，缺失用默认值；password 类型不回显）
func (s *Service) GetSettings(ctx context.Context) ([]SettingTab, error) {
	values := map[string]string{}
	cfgs, err := s.ent.SysConfig.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	for _, c := range cfgs {
		values[c.Key] = c.Value
	}
	out := make([]SettingTab, 0, len(settingsTabs))
	for _, tab := range settingsTabs {
		nt := SettingTab{Key: tab.Key, Title: tab.Title, Sections: make([]SettingSection, 0, len(tab.Sections))}
		for _, sec := range tab.Sections {
			ns := SettingSection{Title: sec.Title, Description: sec.Description, Items: make([]SettingItem, 0, len(sec.Items))}
			for _, item := range sec.Items {
				ni := item
				ni.Value = item.Default
				if v, ok := values[item.Key]; ok && v != "" {
					ni.Value = v
				}
				if item.Type == "password" {
					ni.Value = "" // 安全：不回显密文
				}
				ns.Items = append(ns.Items, ni)
			}
			nt.Sections = append(nt.Sections, ns)
		}
		out = append(out, nt)
	}
	return out, nil
}

// SaveSettings 批量保存（仅接受注册表内的 key；password 留空表示不修改）
func (s *Service) SaveSettings(ctx context.Context, values map[string]string) error {
	registry := map[string]*SettingItem{}
	for _, item := range registryItems() {
		it := item
		registry[it.Key] = &it
	}
	for key, val := range values {
		item, ok := registry[key]
		if !ok {
			continue // 注册表之外的 key 一律忽略
		}
		if item.Type == "password" && val == "" {
			continue // 留空不修改
		}
		val = strings.TrimSpace(val)
		existing, err := s.ent.SysConfig.Query().Where(sysconfig.KeyEQ(key)).Only(ctx)
		if ent.IsNotFound(err) {
			_, err = s.ent.SysConfig.Create().
				SetName(key).SetKey(key).SetValue(val).SetRemark(item.Label).Save(ctx)
			if err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		if existing.Value == val {
			continue
		}
		if err := s.ent.SysConfig.UpdateOneID(existing.ID).SetValue(val).Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

// SendTestMail 用当前 SMTP 配置发送测试邮件
func (s *Service) SendTestMail(ctx context.Context, to string) error {
	if strings.TrimSpace(to) == "" || !strings.Contains(to, "@") {
		return common.NewBizError(http.StatusBadRequest, "收件人邮箱格式不正确")
	}
	mailer := auth.NewMailerFromConfig(ctx, s.ent)
	if mailer == nil {
		return common.NewBizError(http.StatusServiceUnavailable, "邮件服务未配置（请先在邮件设置中填写 SMTP 主机）")
	}
	subject := "【GoKeep】测试邮件"
	html := `<div style="font-family:system-ui,sans-serif;max-width:480px;margin:0 auto;padding:24px;border:1px solid #e2e8f0;border-radius:12px">
<h2 style="margin:0 0 12px;color:#0f172a">GoKeep 测试邮件</h2>
<p style="color:#475569">这是一封 SMTP 配置测试邮件。收到即表示邮件服务配置正确。</p>
</div>`
	if err := mailer.Send(strings.TrimSpace(to), subject, html); err != nil {
		return common.NewBizError(http.StatusServiceUnavailable, "测试邮件发送失败: "+err.Error())
	}
	return nil
}
