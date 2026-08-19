// Package migrations 嵌入并按版本执行参考实现的 SQL Migration。
package migrations

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed *.up.sql *.down.sql
var migrationFS embed.FS

// File 表示一份从内嵌文件系统读取的不可变 Migration。
type File struct {
	Version  int64
	Name     string
	SQL      string
	Checksum string
}

// Files 按版本返回 Migration。每次都创建新切片，避免测试或工具修改进程级数据源。
func Files() ([]File, error) {
	entries, err := fs.Glob(migrationFS, "*.up.sql")
	if err != nil {
		return nil, fmt.Errorf("list embedded migrations: %w", err)
	}

	files := make([]File, 0, len(entries))
	for _, path := range entries {
		base := filepath.Base(path)
		parts := strings.SplitN(strings.TrimSuffix(base, ".up.sql"), "_", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid migration filename %q", base)
		}
		version, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil || version <= 0 {
			return nil, fmt.Errorf("invalid migration version in %q", base)
		}
		contents, err := migrationFS.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read migration %q: %w", base, err)
		}
		files = append(files, File{
			Version:  version,
			Name:     parts[1],
			SQL:      string(contents),
			Checksum: migrationChecksum(contents),
		})
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Version < files[j].Version })
	for i := 1; i < len(files); i++ {
		if files[i-1].Version == files[i].Version {
			return nil, fmt.Errorf("duplicate migration version %d", files[i].Version)
		}
	}
	return files, nil
}

// migrationChecksum 先统一 Git 在不同操作系统可能转换的换行符，再计算不可变摘要。
// 这样同一提交在 Windows 与 Linux 部署时不会误判已执行 Migration 被篡改。
func migrationChecksum(contents []byte) string {
	normalized := bytes.ReplaceAll(contents, []byte("\r\n"), []byte("\n"))
	normalized = bytes.ReplaceAll(normalized, []byte("\r"), []byte("\n"))
	checksum := sha256.Sum256(normalized)
	return hex.EncodeToString(checksum[:])
}

// Apply 执行尚未登记到 schema_migrations 的全部 Migration。
//
// MySQL DDL 会隐式提交，不能伪装成整份文件可事务回滚。因此建表语句必须可重试，并在全部
// 语句成功后才登记版本。命名锁还保证多个迁移进程不会同时通过“未执行”检查。
func Apply(ctx context.Context, db *sql.DB) (resultErr error) {
	if db == nil {
		return fmt.Errorf("apply migrations: database is required")
	}
	connection, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("reserve migration connection: %w", err)
	}
	defer connection.Close()

	const migrationLock = "investment_community_schema_migrations"
	var acquired sql.NullInt64
	if err := connection.QueryRowContext(ctx, "SELECT GET_LOCK(?, 10)", migrationLock).Scan(&acquired); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	if !acquired.Valid || acquired.Int64 != 1 {
		return fmt.Errorf("acquire migration lock: timed out")
	}
	defer func() {
		releaseContext, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		var released sql.NullInt64
		if err := connection.QueryRowContext(releaseContext, "SELECT RELEASE_LOCK(?)", migrationLock).Scan(&released); err != nil {
			if resultErr == nil {
				resultErr = fmt.Errorf("release migration lock: %w", err)
			}
			return
		}
		if (!released.Valid || released.Int64 != 1) && resultErr == nil {
			resultErr = fmt.Errorf("release migration lock: lock was not owned")
		}
	}()

	if _, err := connection.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version BIGINT NOT NULL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    checksum CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    applied_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	files, err := Files()
	if err != nil {
		return err
	}
	for _, file := range files {
		var appliedChecksum string
		err := connection.QueryRowContext(ctx,
			"SELECT checksum FROM schema_migrations WHERE version = ?", file.Version,
		).Scan(&appliedChecksum)
		if err == nil {
			if appliedChecksum != file.Checksum {
				return fmt.Errorf("migration %d checksum mismatch", file.Version)
			}
			if file.Version == 1 {
				if err := verifyInitialSchema(ctx, connection); err != nil {
					return err
				}
			}
			continue
		}
		if err != sql.ErrNoRows {
			return fmt.Errorf("check migration %d: %w", file.Version, err)
		}

		for _, statement := range splitStatements(file.SQL) {
			if _, err := connection.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("apply migration %03d_%s: %w", file.Version, file.Name, err)
			}
		}
		if file.Version == 1 {
			if err := verifyInitialSchema(ctx, connection); err != nil {
				return err
			}
		}
		if _, err := connection.ExecContext(ctx,
			"INSERT INTO schema_migrations (version, name, checksum) VALUES (?, ?, ?)",
			file.Version, file.Name, file.Checksum,
		); err != nil {
			return fmt.Errorf("record migration %d: %w", file.Version, err)
		}
	}
	return nil
}

func verifyInitialSchema(ctx context.Context, connection *sql.Conn) error {
	markers := []string{
		"chk_users_schema_v1",
		"chk_circles_schema_v1",
		"chk_circle_memberships_schema_v1",
		"chk_securities_schema_v1",
		"chk_posts_schema_v1",
		"chk_post_securities_schema_v1",
		"chk_comments_schema_v1",
		"chk_reports_schema_v1",
		"chk_notifications_schema_v1",
		"chk_admin_audit_schema_v1",
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(markers)), ",")
	arguments := make([]any, len(markers))
	for index, marker := range markers {
		arguments[index] = marker
	}
	query := `SELECT COUNT(*)
FROM information_schema.table_constraints
WHERE constraint_schema = DATABASE()
  AND constraint_type = 'CHECK'
  AND constraint_name IN (` + placeholders + `)`
	var found int
	if err := connection.QueryRowContext(ctx, query, arguments...).Scan(&found); err != nil {
		return fmt.Errorf("verify initial schema markers: %w", err)
	}
	if found != len(markers) {
		return fmt.Errorf("verify initial schema markers: found %d of %d; reset or repair incompatible pre-existing tables", found, len(markers))
	}
	return nil
}

func splitStatements(source string) []string {
	parts := strings.Split(source, ";")
	statements := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		statements = append(statements, part)
	}
	return statements
}
