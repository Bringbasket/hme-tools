// Redis key 规范（docs/03 §7：gokeep: 前缀）
package common

const (
	KeySessionPrefix = "gokeep:session:"   // 会话：<jti> -> userId，TTL=登录有效期
	KeyCaptchaPrefix = "gokeep:captcha:"   // 验证码：<uuid> -> 答案，TTL=5min
	KeyPermsPrefix   = "gokeep:userperms:" // 权限缓存：<userId> -> {"isAdmin":bool,"perms":[]}
	// 邮箱验证码（注册/找回）：<purpose>:<email> -> {code,attempts,expiresAt}，TTL=15min
	KeyEmailCodePrefix = "gokeep:emailcode:"
	// 邮箱验证码发送冷却：<purpose>:<email> -> 1，TTL=60s
	KeyEmailCodeCooldownPrefix = "gokeep:emailcode:cooldown:"
)
