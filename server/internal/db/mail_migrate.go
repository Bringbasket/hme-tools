package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"gokeep/server/internal/ent"
)

//go:embed migrations/*.up.sql
var mailMigrationFiles embed.FS

// migrateMailSchema creates the mail module's tables in the same PostgreSQL
// database as GoKeep. Statements are intentionally idempotent so existing
// installations can be upgraded without a separate migration table.
func migrateMailSchema(ctx context.Context, client *ent.Client) error {
	db := client.SQLDB()
	if db == nil {
		return fmt.Errorf("mail schema requires an SQL database driver")
	}
	entries, err := fs.Glob(mailMigrationFiles, "migrations/*.up.sql")
	if err != nil {
		return fmt.Errorf("list mail migrations: %w", err)
	}
	sort.Strings(entries)
	for _, name := range entries {
		contents, readErr := mailMigrationFiles.ReadFile(name)
		if readErr != nil {
			return fmt.Errorf("read mail migration %s: %w", name, readErr)
		}
		if _, execErr := db.ExecContext(ctx, strings.TrimSpace(string(contents))); execErr != nil {
			return fmt.Errorf("apply mail migration %s: %w", name, execErr)
		}
	}
	return nil
}
