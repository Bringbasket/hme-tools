// 邮件服务：SMTP 配置读取自参数配置（sys_config），未配置时视为"邮件服务未启用"
// 注册邮件发送器。
package auth

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"

	"gokeep/server/internal/ent"
	"gokeep/server/internal/ent/sysconfig"
)

// Mailer SMTP 邮件发送器
type Mailer struct {
	host     string
	port     int
	username string
	password string
	from     string
	useSSL   bool
}

// ConfigValue 读取参数配置（key 缺失返回空）
func ConfigValue(ctx context.Context, client *ent.Client, key string) string {
	cfg, err := client.SysConfig.Query().Where(sysconfig.KeyEQ(key)).Only(ctx)
	if err != nil {
		return ""
	}
	return cfg.Value
}

// NewMailerFromConfig 从 sys_config 装配邮件发送器；host 为空返回 nil（未配置）
func NewMailerFromConfig(ctx context.Context, client *ent.Client) *Mailer {
	host := strings.TrimSpace(ConfigValue(ctx, client, "sys.mail.host"))
	if host == "" {
		return nil
	}
	port := 465
	if v := ConfigValue(ctx, client, "sys.mail.port"); v != "" {
		fmt.Sscanf(v, "%d", &port)
	}
	useSSL := strings.EqualFold(ConfigValue(ctx, client, "sys.mail.ssl"), "true")
	return &Mailer{
		host:     host,
		port:     port,
		username: ConfigValue(ctx, client, "sys.mail.username"),
		password: ConfigValue(ctx, client, "sys.mail.password"),
		from:     ConfigValue(ctx, client, "sys.mail.from"),
		useSSL:   useSSL,
	}
}

// Send 发送邮件；useSSL=true 走 SSL 直连（465），否则 smtp.SendMail（587 STARTTLS）
func (m *Mailer) Send(to, subject, htmlBody string) error {
	if m == nil || m.host == "" {
		return fmt.Errorf("邮件服务未配置")
	}
	from := m.from
	if from == "" {
		from = m.username
	}
	if from == "" {
		return fmt.Errorf("发件人未配置（sys.mail.from）")
	}
	header := fmt.Sprintf(
		"From: GoKeep <%s>\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n",
		from, to, subject,
	)
	msg := []byte(header + htmlBody)
	addr := fmt.Sprintf("%s:%d", m.host, m.port)
	if m.useSSL {
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: m.host})
		if err != nil {
			return fmt.Errorf("SMTP SSL 连接失败: %w", err)
		}
		defer conn.Close()
		client, err := smtp.NewClient(conn, m.host)
		if err != nil {
			return fmt.Errorf("SMTP 会话失败: %w", err)
		}
		defer client.Close()
		return smtpSend(client, m, from, []string{to}, msg)
	}
	auth := smtp.PlainAuth("", m.username, m.password, m.host)
	if m.username == "" {
		auth = nil
	}
	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}

func smtpSend(client *smtp.Client, m *Mailer, from string, to []string, msg []byte) error {
	if m.username != "" {
		if err := client.Auth(smtp.PlainAuth("", m.username, m.password, m.host)); err != nil {
			return fmt.Errorf("SMTP 认证失败: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("SMTP MAIL 失败: %w", err)
	}
	for _, addr := range to {
		if err := client.Rcpt(addr); err != nil {
			return fmt.Errorf("SMTP RCPT 失败: %w", err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA 失败: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("SMTP 写入失败: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("SMTP 发送失败: %w", err)
	}
	return client.Quit()
}
