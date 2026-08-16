// 备份服务：手动/定时创建全库备份 → gzip → S3；记录表管理；下载/恢复/删除
package backup

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"gokeep/server/internal/common"
	"gokeep/server/internal/ent"
	"gokeep/server/internal/ent/backuprecord"
)

type Service struct {
	ent *ent.Client
	db  *sql.DB
}

// New 创建备份服务（独立 DB 连接用于 dump/恢复；databaseURL 同主库）
func New(client *ent.Client, databaseURL string) *Service {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		db = nil
	}
	return &Service{ent: client, db: db}
}

// Ent 暴露 ent client（供 handler 读取配置）
func (s *Service) Ent() *ent.Client {
	return s.ent
}

// DB 备份专用 sql.DB
func (s *Service) DB() *sql.DB {
	return s.db
}

// ==================== 创建备份 ====================

// CreateBackup 立即创建备份（异步执行 dump+上传，返回记录）
func (s *Service) CreateBackup(ctx context.Context, triggerType string, expireDays int) (*ent.BackupRecord, error) {
	cfg := LoadS3Config(ctx, s.ent)
	if !cfg.configured() {
		return nil, common.NewBizError(http.StatusServiceUnavailable, "S3 存储未配置，请先在数据备份里填写并保存")
	}
	if err := cfg.TestConnection(ctx); err != nil {
		return nil, err
	}
	now := time.Now()
	key := randomKey()
	fname := fmt.Sprintf("gokeep_%s_%s.sql.gz", now.Format("20060102_150405"), key)
	var expire *time.Time
	if expireDays > 0 {
		t := now.AddDate(0, 0, expireDays)
		expire = &t
	}
	rec, err := s.ent.BackupRecord.Create().
		SetRecordKey(key).SetStatus("running").SetFileName(fname).SetParts(1).
		SetTriggerType(triggerType).SetStartedAt(now).SetNillableExpireAt(expire).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	go s.runBackup(rec.ID)
	return rec, nil
}

// runBackup 后台执行：dump → gzip → 上传 → 更新记录
func (s *Service) runBackup(recordID int64) {
	ctx := context.Background()
	start := time.Now()
	rec, err := s.ent.BackupRecord.Get(ctx, recordID)
	if err != nil {
		return
	}
	db := s.DB()
	if db == nil {
		s.fail(ctx, recordID, "数据库连接不可用")
		return
	}
	cfg := LoadS3Config(ctx, s.ent)

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if err := DumpDatabase(ctx, db, gz); err != nil {
		s.fail(ctx, recordID, err.Error())
		return
	}
	if err := gz.Close(); err != nil {
		s.fail(ctx, recordID, err.Error())
		return
	}

	if err := cfg.Upload(ctx, rec.FileName, bytes.NewReader(buf.Bytes()), int64(buf.Len())); err != nil {
		s.fail(ctx, recordID, err.Error())
		return
	}
	now := time.Now()
	_, _ = s.ent.BackupRecord.UpdateOneID(recordID).
		SetStatus("completed").SetSizeBytes(int64(buf.Len())).
		SetFinishedAt(now).SetDurationMs(float64(now.Sub(start).Milliseconds())).
		ClearErrorMessage().Save(ctx)
}

func (s *Service) fail(ctx context.Context, id int64, msg string) {
	now := time.Now()
	_, _ = s.ent.BackupRecord.UpdateOneID(id).
		SetStatus("failed").SetErrorMessage(msg).SetFinishedAt(now).
		SetDurationMs(float64(now.UnixMilli())).Save(ctx)
}

// ==================== 记录管理 ====================

type RecordView struct {
	ID          int64   `json:"id"`
	RecordKey   string  `json:"recordKey"`
	Status      string  `json:"status"`
	FileName    string  `json:"fileName"`
	SizeBytes   int64   `json:"sizeBytes"`
	Parts       int     `json:"parts"`
	ExpireAt    *string `json:"expireAt"`
	TriggerType string  `json:"triggerType"`
	StartedAt   string  `json:"startedAt"`
	FinishedAt  *string `json:"finishedAt"`
	DurationMs  float64 `json:"durationMs"`
	Error       *string `json:"error"`
	CreatedAt   string  `json:"createdAt"`
}

func toView(r *ent.BackupRecord) RecordView {
	v := RecordView{
		ID: r.ID, RecordKey: r.RecordKey, Status: r.Status, FileName: r.FileName,
		SizeBytes: r.SizeBytes, Parts: r.Parts, TriggerType: r.TriggerType,
		StartedAt: r.StartedAt.Format("2006-01-02 15:04:05"), DurationMs: r.DurationMs,
		CreatedAt: r.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if r.ExpireAt != nil {
		t := r.ExpireAt.Format("2006-01-02 15:04:05")
		v.ExpireAt = &t
	}
	if r.FinishedAt != nil {
		t := r.FinishedAt.Format("2006-01-02 15:04:05")
		v.FinishedAt = &t
	}
	if r.ErrorMessage != nil && *r.ErrorMessage != "" {
		v.Error = r.ErrorMessage
	}
	return v
}

func (s *Service) ListRecords(ctx context.Context, pq common.PageQuery) (*common.PageResult[RecordView], error) {
	q := s.ent.BackupRecord.Query()
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	recs, err := q.Order(ent.Desc(backuprecord.FieldID)).Offset(pq.Offset).Limit(pq.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]RecordView, 0, len(recs))
	for _, r := range recs {
		list = append(list, toView(r))
	}
	return &common.PageResult[RecordView]{List: list, Total: int64(total), Page: pq.Page, PageSize: pq.PageSize}, nil
}

// DownloadRecord 下载备份文件
func (s *Service) DownloadRecord(ctx context.Context, id int64) (io.ReadCloser, string, error) {
	rec, err := s.ent.BackupRecord.Get(ctx, id)
	if err != nil {
		return nil, "", common.NewBizError(http.StatusNotFound, "备份记录不存在")
	}
	if rec.Status != "completed" {
		return nil, "", common.NewBizError(http.StatusBadRequest, "备份尚未完成，无法下载")
	}
	cfg := LoadS3Config(ctx, s.ent)
	obj, err := cfg.Download(ctx, rec.FileName)
	if err != nil {
		return nil, "", err
	}
	return obj, rec.FileName, nil
}

// DeleteRecord 删除备份（S3 对象 + 记录）
func (s *Service) DeleteRecord(ctx context.Context, id int64) error {
	rec, err := s.ent.BackupRecord.Get(ctx, id)
	if err != nil {
		return common.NewBizError(http.StatusNotFound, "备份记录不存在")
	}
	if rec.Status == "completed" {
		cfg := LoadS3Config(ctx, s.ent)
		if cfg.configured() {
			_ = cfg.Delete(ctx, rec.FileName) // 对象删除失败不阻塞记录清理
		}
	}
	return s.ent.BackupRecord.DeleteOneID(id).Exec(ctx)
}

// RestoreRecord 恢复备份：清空全库 → 执行备份 SQL
func (s *Service) RestoreRecord(ctx context.Context, id int64) error {
	rec, err := s.ent.BackupRecord.Get(ctx, id)
	if err != nil {
		return common.NewBizError(http.StatusNotFound, "备份记录不存在")
	}
	if rec.Status != "completed" {
		return common.NewBizError(http.StatusBadRequest, "备份尚未完成，无法恢复")
	}
	cfg := LoadS3Config(ctx, s.ent)
	obj, err := cfg.Download(ctx, rec.FileName)
	if err != nil {
		return err
	}
	defer obj.Close()
	gz, err := gzip.NewReader(obj)
	if err != nil {
		return fmt.Errorf("解压备份失败: %w", err)
	}
	defer gz.Close()
	raw, err := io.ReadAll(gz)
	if err != nil {
		return fmt.Errorf("读取备份失败: %w", err)
	}
	db := s.DB()
	if db == nil {
		return fmt.Errorf("数据库连接不可用")
	}
	// 事务包裹：清空 + 重放任一步失败整体回滚，避免半截数据
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开启恢复事务失败: %w", err)
	}
	defer tx.Rollback()
	if err := TruncateAll(ctx, tx); err != nil {
		return fmt.Errorf("清空现有数据失败: %w", err)
	}
	if err := RestoreSQL(ctx, tx, string(raw)); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("恢复提交失败: %w", err)
	}
	return nil
}

// ==================== 定时调度 ====================

// ScheduleConfig 定时备份配置
type ScheduleConfig struct {
	Enabled    bool
	Cron       string
	ExpireDays int
	MaxKeep    int
}

// LoadScheduleConfig 读取定时备份配置
func LoadScheduleConfig(ctx context.Context, client *ent.Client) ScheduleConfig {
	c := ScheduleConfig{
		Enabled: ConfigValue(ctx, client, "sys.backup.schedule.enabled") == "true",
		Cron:    ConfigValue(ctx, client, "sys.backup.schedule.cron"),
		ExpireDays: atoiDefault(ConfigValue(ctx, client, "sys.backup.schedule.expireDays"), 0),
		MaxKeep:    atoiDefault(ConfigValue(ctx, client, "sys.backup.schedule.maxKeep"), 0),
	}
	if c.Cron == "" {
		c.Cron = "0 2 * * *"
	}
	return c
}

// CleanupExpired 清理过期/超量备份（过期天数 + 最大保留份数）
func (s *Service) CleanupExpired(ctx context.Context) {
	cfg := LoadScheduleConfig(ctx, s.ent)
	recs, err := s.ent.BackupRecord.Query().Order(ent.Desc(backuprecord.FieldID)).All(ctx)
	if err != nil {
		return
	}
	now := time.Now()
	deleted := 0
	kept := 0
	for _, r := range recs {
		shouldDelete := false
		if cfg.ExpireDays > 0 && r.ExpireAt != nil && r.ExpireAt.Before(now) {
			shouldDelete = true
		}
		if cfg.MaxKeep > 0 && kept >= cfg.MaxKeep {
			shouldDelete = true
		}
		if shouldDelete {
			if r.Status == "completed" {
				s3cfg := LoadS3Config(ctx, s.ent)
				if s3cfg.configured() {
					_ = s3cfg.Delete(ctx, r.FileName)
				}
			}
			_ = s.ent.BackupRecord.DeleteOneID(r.ID).Exec(ctx)
			deleted++
			continue
		}
		kept++
	}
	_ = deleted
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func randomKey() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
