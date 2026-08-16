package backup

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// TestRestoreRoundtrip 全库 dump → truncate → restore，校验 admin 密码哈希不变
func TestRestoreRoundtrip(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	db, _ := sql.Open("pgx", url)
	defer db.Close()

	hashBefore := queryHash(t, db)

	var sb strings.Builder
	if err := DumpDatabase(context.Background(), db, &sb); err != nil {
		t.Fatal(err)
	}
	sqlText := sb.String()

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := TruncateAll(context.Background(), tx); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := RestoreSQL(context.Background(), tx, sqlText); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	hashAfter := queryHash(t, db)
	if hashBefore != hashAfter {
		t.Errorf("password hash changed after restore roundtrip")
	}
}

func queryHash(t *testing.T, db *sql.DB) string {
	var h string
	if err := db.QueryRow(`SELECT password FROM sys_users WHERE username='admin'`).Scan(&h); err != nil {
		t.Fatalf("query admin: %v", err)
	}
	return h
}
