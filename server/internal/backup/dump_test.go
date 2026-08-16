package backup

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"io"
	"os"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// TestDumpDatabase 只读校验：导出本地库并确认含关键表与 INSERT 语句
// 仅当 DATABASE_URL 已配置时运行（否则跳过）
func TestDumpDatabase(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set, skip")
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Skipf("db unreachable: %v", err)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if err := DumpDatabase(context.Background(), db, gz); err != nil {
		t.Fatalf("dump: %v", err)
	}
	gz.Close()
	compressedSize := buf.Len() // 在读取前记录压缩后大小

	r, err := gzip.NewReader(&buf)
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	raw, _ := io.ReadAll(r)
	r.Close()
	sqlText := string(raw)

	if !strings.Contains(sqlText, "sys_users") {
		t.Errorf("dump missing sys_users table")
	}
	if !strings.Contains(sqlText, "INSERT INTO") {
		t.Errorf("dump missing INSERT statements")
	}
	if !strings.Contains(sqlText, "admin") {
		t.Errorf("dump missing admin user row")
	}
	if compressedSize == 0 {
		t.Errorf("compressed dump is empty")
	}
	t.Logf("dump compressed=%d bytes, sql=%d bytes", compressedSize, len(sqlText))
}
