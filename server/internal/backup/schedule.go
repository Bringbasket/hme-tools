// 定时备份调度：按 Cron 表达式触发备份 + 过期清理（每次触发时重读配置，改配置即时生效）
package backup

import (
	"context"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
)

// StartScheduler 启动后台调度循环（阻塞，以 goroutine 调用）
func (s *Service) StartScheduler(ctx context.Context) {
	go func() {
		for {
			cfg := LoadScheduleConfig(ctx, s.ent)
			if !cfg.Enabled {
				time.Sleep(30 * time.Second)
				continue
			}
			schedule, err := cron.ParseStandard(cfg.Cron)
			if err != nil {
				slog.Error("backup scheduler: cron 表达式无效", "cron", cfg.Cron, "err", err)
				time.Sleep(5 * time.Minute)
				continue
			}
			next := schedule.Next(time.Now())
			slog.Info("backup scheduler: 下次备份", "at", next.Format(time.RFC3339))
			timer := time.NewTimer(time.Until(next))
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				// 触发前重新读配置（可能已关闭/已改表达式）
				live := LoadScheduleConfig(ctx, s.ent)
				if !live.Enabled {
					continue
				}
				if _, err := s.CreateBackup(ctx, "scheduled", live.ExpireDays); err != nil {
					slog.Error("backup scheduler: 备份失败", "err", err)
				} else {
					slog.Info("backup scheduler: 定时备份已触发")
				}
				s.CleanupExpired(ctx)
			}
		}
	}()
}
