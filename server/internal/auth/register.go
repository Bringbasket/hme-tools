// 注册服务：邮箱注册 + 可选邮箱验证码（开关 sys.register.emailCodeEnabled）
// 邮箱注册：验证码 TTL 15 分钟、发送冷却 60 秒、最多 5 次尝试。
package auth

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"gokeep/server/internal/common"
	"gokeep/server/internal/ent/sysconfig"
	"gokeep/server/internal/ent/sysrole"
	"gokeep/server/internal/ent/sysuser"
	"gokeep/server/internal/ent/sysuserrole"
)

const (
	emailCodeTTL             = 15 * time.Minute // 验证码有效期
	emailCodeCooldown        = 60 * time.Second // 发送冷却
	emailCodeMaxAttempts     = 5                // 最多尝试次数
	emailCodePurposeRegister = "register"
)

// RegisterConfig 注册开关（前端据此渲染表单）
type RegisterConfig struct {
	RegisterEnabled  bool `json:"registerEnabled"`
	EmailCodeEnabled bool `json:"emailCodeEnabled"`
}

// RegisterReq 注册请求
type RegisterReq struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	Nickname string `json:"nickname"`
	Code     string `json:"code"` // 邮箱验证码（开关开启时必填）
}

type emailCodeRecord struct {
	Code      string    `json:"code"`
	Attempts  int       `json:"attempts"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// GetRegisterConfig 注册开关状态
func (s *Service) GetRegisterConfig(ctx context.Context) RegisterConfig {
	return RegisterConfig{
		RegisterEnabled:  s.configBool(ctx, "sys.account.registerUser", false),
		EmailCodeEnabled: s.configBool(ctx, "sys.register.emailCodeEnabled", false),
	}
}

// SendRegisterEmailCode 发送注册邮箱验证码；开发环境未配置 SMTP 时返回 devCode 便于无头测试
func (s *Service) SendRegisterEmailCode(ctx context.Context, email string) (string, error) {
	cfg := s.GetRegisterConfig(ctx)
	if !cfg.RegisterEnabled {
		return "", common.NewBizError(http.StatusForbidden, "注册暂未开放")
	}
	if !cfg.EmailCodeEnabled {
		return "", common.NewBizError(http.StatusBadRequest, "邮箱验证码未开启")
	}
	email, err := normalizeEmail(email)
	if err != nil {
		return "", common.NewBizError(http.StatusBadRequest, "邮箱格式不正确")
	}

	// 发送冷却（防轰炸）
	cooldownKey := common.KeyEmailCodeCooldownPrefix + emailCodePurposeRegister + ":" + email
	ok, err := s.rdb.SetNX(ctx, cooldownKey, "1", emailCodeCooldown).Result()
	if err != nil {
		return "", err
	}
	if !ok {
		return "", common.NewBizError(http.StatusTooManyRequests, "发送太频繁，请 1 分钟后再试")
	}

	code, err := randomDigits(6)
	if err != nil {
		return "", err
	}
	record := emailCodeRecord{Code: code, Attempts: 0, ExpiresAt: time.Now().Add(emailCodeTTL)}
	raw, _ := json.Marshal(record)
	codeKey := common.KeyEmailCodePrefix + emailCodePurposeRegister + ":" + email
	if err := s.rdb.Set(ctx, codeKey, raw, emailCodeTTL).Err(); err != nil {
		return "", err
	}

	mailer := NewMailerFromConfig(ctx, s.ent)
	if mailer == nil {
		// 未配置 SMTP：开发环境返回 devCode（对齐验证码 expr 的无头测试约定），生产报错
		if s.env == "development" {
			return code, nil
		}
		_ = s.rdb.Del(ctx, codeKey).Err()
		return "", common.NewBizError(http.StatusServiceUnavailable, "邮件服务未配置，请联系管理员")
	}
	subject := "【GoKeep】注册验证码"
	body := fmt.Sprintf(
		`<div style="font-family:system-ui,sans-serif;max-width:480px;margin:0 auto;padding:24px;border:1px solid #e2e8f0;border-radius:12px">
<h2 style="margin:0 0 12px;color:#0f172a">GoKeep 邮箱验证</h2>
<p style="color:#475569">您的注册验证码为（%d 分钟内有效）：</p>
<p style="font-size:28px;font-weight:700;letter-spacing:6px;color:#2563eb">%s</p>
<p style="color:#94a3b8;font-size:12px">如非本人操作，请忽略本邮件。</p>
</div>`,
		int(emailCodeTTL.Minutes()), code,
	)
	if err := mailer.Send(email, subject, body); err != nil {
		_ = s.rdb.Del(ctx, codeKey).Err()
		return "", common.NewBizError(http.StatusServiceUnavailable, "验证码发送失败，请稍后重试")
	}
	return "", nil
}

// Register 邮箱注册；成功即登录（返回 token）。
func (s *Service) Register(ctx context.Context, req RegisterReq) (token string, userID int64, username, nickname string, parentUserID int64, err error) {
	cfg := s.GetRegisterConfig(ctx)
	if !cfg.RegisterEnabled {
		return "", 0, "", "", 0, common.NewBizError(http.StatusForbidden, "注册暂未开放")
	}
	email, err := normalizeEmail(req.Email)
	if err != nil {
		return "", 0, "", "", 0, common.NewBizError(http.StatusBadRequest, "邮箱格式不正确")
	}
	if len(req.Password) < 8 {
		return "", 0, "", "", 0, common.NewBizError(http.StatusBadRequest, "密码至少 8 位")
	}

	// 邮箱唯一（username=email）
	exists, err := s.ent.SysUser.Query().Where(sysuser.UsernameEQ(email)).Exist(ctx)
	if err != nil {
		return "", 0, "", "", 0, err
	}
	if exists {
		return "", 0, "", "", 0, common.NewBizError(http.StatusConflict, "该邮箱已注册，请直接登录")
	}

	// 验证码（开关开启时必填）
	if cfg.EmailCodeEnabled {
		if err := s.verifyEmailCode(ctx, email, req.Code); err != nil {
			return "", 0, "", "", 0, err
		}
	}

	hash, err := common.HashPassword(req.Password)
	if err != nil {
		return "", 0, "", "", 0, err
	}
	nickname = strings.TrimSpace(req.Nickname)
	if nickname == "" {
		nickname = strings.Split(email, "@")[0]
	}
	user, err := s.ent.SysUser.Create().
		SetUsername(email).SetEmail(email).SetPassword(hash).SetNickname(nickname).SetStatus("0").Save(ctx)
	if err != nil {
		return "", 0, "", "", 0, err
	}
	// 绑定普通用户角色（seed 保证存在）
	if role, rerr := s.ent.SysRole.Query().Where(sysrole.CodeEQ("common")).Only(ctx); rerr == nil {
		n, _ := s.ent.SysUserRole.Query().
			Where(sysuserrole.UserID(user.ID), sysuserrole.RoleID(role.ID)).Count(ctx)
		if n == 0 {
			_, _ = s.ent.SysUserRole.Create().SetUserID(user.ID).SetRoleID(role.ID).Save(ctx)
		}
	}

	token, err = s.issueSession(ctx, user.ID)
	if err != nil {
		return "", 0, "", "", 0, err
	}
	return token, user.ID, user.Username, user.Nickname, parentUserID, nil
}

// verifyEmailCode 校验邮箱验证码（错 1 次 attempts+1，超 5 次作废重发）
func (s *Service) verifyEmailCode(ctx context.Context, email, code string) error {
	if strings.TrimSpace(code) == "" {
		return common.NewBizError(http.StatusBadRequest, "请输入邮箱验证码")
	}
	key := common.KeyEmailCodePrefix + emailCodePurposeRegister + ":" + email
	raw, err := s.rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return common.NewBizError(http.StatusBadRequest, "验证码已过期，请重新获取")
	}
	if err != nil {
		return err
	}
	var record emailCodeRecord
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return err
	}
	if record.Attempts >= emailCodeMaxAttempts {
		_ = s.rdb.Del(ctx, key).Err()
		return common.NewBizError(http.StatusTooManyRequests, "尝试次数过多，请重新获取验证码")
	}
	if record.Code != strings.TrimSpace(code) {
		record.Attempts++
		if record.Attempts >= emailCodeMaxAttempts {
			_ = s.rdb.Del(ctx, key).Err()
			return common.NewBizError(http.StatusTooManyRequests, "尝试次数过多，请重新获取验证码")
		}
		ttl := time.Until(record.ExpiresAt)
		if ttl <= 0 {
			_ = s.rdb.Del(ctx, key).Err()
			return common.NewBizError(http.StatusBadRequest, "验证码已过期，请重新获取")
		}
		updated, err := json.Marshal(record)
		if err != nil {
			return err
		}
		_ = s.rdb.Set(ctx, key, updated, ttl).Err()
		remaining := emailCodeMaxAttempts - record.Attempts
		return common.NewBizError(http.StatusBadRequest, fmt.Sprintf("验证码错误（剩余 %d 次尝试）", remaining))
	}
	_ = s.rdb.Del(ctx, key).Err()
	return nil
}

func (s *Service) configBool(ctx context.Context, key string, def bool) bool {
	cfg, err := s.ent.SysConfig.Query().Where(sysconfig.KeyEQ(key)).Only(ctx)
	if err != nil {
		return def
	}
	return strings.EqualFold(cfg.Value, "true")
}

func normalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || !strings.Contains(email, "@") {
		return "", fmt.Errorf("invalid email")
	}
	return email, nil
}

func randomDigits(n int) (string, error) {
	out := make([]byte, n)
	for i := range out {
		v, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		out[i] = byte('0' + v.Int64())
	}
	return string(out), nil
}
