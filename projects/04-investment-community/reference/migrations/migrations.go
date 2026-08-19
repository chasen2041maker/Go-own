// Package migrations 嵌入并按版本执行参考实现的 SQL Migration。
package migrations

import (
	"context"
	"database/sql"
	"embed"
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
	Version int64
	Name    string
	SQL     string
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
		files = append(files, File{Version: version, Name: parts[1], SQL: string(contents)})
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Version < files[j].Version })
	for i := 1; i < len(files); i++ {
		if files[i-1].Version == files[i].Version {
			return nil, fmt.Errorf("duplicate migration version %d", files[i].Version)
		}
	}
	return files, nil
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
    version BIGINT UNSIGNED NOT NULL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    applied_at DATETIME(6) NOT NULL DEFAULT (UTC_TIMESTAMP(6))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	files, err := Files()
	if err != nil {
		return err
	}
	for _, file := range files {
		var exists int
		err := connection.QueryRowContext(ctx, "SELECT 1 FROM schema_migrations WHERE version = ?", file.Version).Scan(&exists)
		if err == nil {
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
		if _, err := connection.ExecContext(ctx,
			"INSERT INTO schema_migrations (version, name) VALUES (?, ?)", file.Version, file.Name,
		); err != nil {
			return fmt.Errorf("record migration %d: %w", file.Version, err)
		}
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
