//go:build integration

package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"go-own/projects/04-investment-community/reference/internal/domain"
	"go-own/projects/04-investment-community/reference/internal/platform"
	"go-own/projects/04-investment-community/reference/migrations"
)

func TestSeedIsIdempotentAndStoresExistingHasherOutput(t *testing.T) {
	dsn := os.Getenv("COMMUNITY_TEST_DSN")
	if dsn == "" {
		t.Skip("COMMUNITY_TEST_DSN is not set")
	}
	database, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := migrations.Apply(ctx, database); err != nil {
		t.Fatalf("migrations.Apply() error = %v", err)
	}

	suffix := strconv.FormatInt(time.Now().UnixNano(), 36)
	if len(suffix) > 8 {
		suffix = suffix[len(suffix)-8:]
	}
	adminPassword := "admin-password-" + suffix
	userPassword := "learner-password-" + suffix
	data := seedData{
		Users: []seedUser{
			{Email: "seed-admin-" + suffix + "@example.test", DisplayName: "种子管理员" + suffix, Password: adminPassword, Role: domain.RoleAdmin},
			{Email: "seed-user-" + suffix + "@example.test", DisplayName: "种子学习者" + suffix, Password: userPassword, Role: domain.RoleUser},
		},
		Securities: []seedSecurity{
			{Exchange: "X" + suffix, Code: "A" + suffix, Name: "虚构证券甲" + suffix, Status: domain.SecurityStatusActive},
			{Exchange: "X" + suffix, Code: "Z" + suffix, Name: "虚构证券乙" + suffix, Status: domain.SecurityStatusInactive},
		},
		Circles: []seedCircle{{Slug: "seed-" + suffix, Name: "虚构种子圈" + suffix, Description: "集成测试虚构数据"}},
	}
	t.Cleanup(func() {
		for _, circle := range data.Circles {
			_, _ = database.Exec("DELETE FROM circles WHERE slug = ?", circle.Slug)
		}
		for _, security := range data.Securities {
			_, _ = database.Exec("DELETE FROM securities WHERE market = ? AND code = ?", security.Exchange, security.Code)
		}
		for _, user := range data.Users {
			_, _ = database.Exec("DELETE FROM users WHERE email = ?", user.Email)
		}
	})

	hasher := platform.NewPasswordHasher()
	for run := 1; run <= 2; run++ {
		if err := applySeedAtomically(ctx, database, hasher, data); err != nil {
			t.Fatalf("applySeedAtomically(run %d) error = %v", run, err)
		}
	}
	for _, user := range data.Users {
		assertSeedRowCount(t, ctx, database, "SELECT COUNT(*) FROM users WHERE email = ?", user.Email)
		var hash string
		if err := database.QueryRowContext(ctx, "SELECT password_hash FROM users WHERE email = ?", user.Email).Scan(&hash); err != nil {
			t.Fatalf("read password hash: %v", err)
		}
		if hash == user.Password || hasher.Verify(hash, user.Password) != nil {
			t.Fatalf("user %s password was not stored through PasswordHasher", user.Email)
		}
	}
	for _, security := range data.Securities {
		assertSeedRowCount(t, ctx, database,
			"SELECT COUNT(*) FROM securities WHERE market = ? AND code = ?", security.Exchange, security.Code)
	}
	for _, circle := range data.Circles {
		assertSeedRowCount(t, ctx, database, "SELECT COUNT(*) FROM circles WHERE slug = ?", circle.Slug)
	}
}

func TestSeedTransactionRollsBackEarlierWritesWhenCatalogWriteFails(t *testing.T) {
	dsn := os.Getenv("COMMUNITY_TEST_DSN")
	if dsn == "" {
		t.Skip("COMMUNITY_TEST_DSN is not set")
	}
	database, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := migrations.Apply(ctx, database); err != nil {
		t.Fatalf("migrations.Apply() error = %v", err)
	}
	suffix := strconv.FormatInt(time.Now().UnixNano(), 36)
	if len(suffix) > 8 {
		suffix = suffix[len(suffix)-8:]
	}
	conflictingName := "回滚冲突圈" + suffix
	preexistingSlug := "rollback-existing-" + suffix
	if _, err := database.ExecContext(ctx,
		"INSERT INTO circles (slug, name, description, status) VALUES (?, ?, '', 'active')",
		preexistingSlug, conflictingName,
	); err != nil {
		t.Fatalf("insert conflicting circle: %v", err)
	}
	t.Cleanup(func() { _, _ = database.Exec("DELETE FROM circles WHERE slug = ?", preexistingSlug) })

	email := "rollback-" + suffix + "@example.test"
	market := "R" + suffix
	code := "R" + suffix
	data := seedData{
		Users:      []seedUser{{Email: email, DisplayName: "回滚用户" + suffix, Password: "rollback-password-123", Role: domain.RoleUser}},
		Securities: []seedSecurity{{Exchange: market, Code: code, Name: "回滚证券" + suffix, Status: domain.SecurityStatusActive}},
		Circles:    []seedCircle{{Slug: "rollback-new-" + suffix, Name: conflictingName, Description: "名称冲突触发最后一步失败"}},
	}
	if err := applySeedAtomically(ctx, database, platform.NewPasswordHasher(), data); err == nil {
		t.Fatal("applySeedAtomically() error = nil, want final circle unique conflict")
	}
	assertNoSeedRow(t, ctx, database, "SELECT COUNT(*) FROM users WHERE email = ?", email)
	assertNoSeedRow(t, ctx, database,
		"SELECT COUNT(*) FROM securities WHERE market = ? AND code = ?", market, code)
}

func assertNoSeedRow(t *testing.T, ctx context.Context, database *sql.DB, query string, arguments ...any) {
	t.Helper()
	var count int
	if err := database.QueryRowContext(ctx, query, arguments...).Scan(&count); err != nil {
		t.Fatalf("count rollback row: %v", err)
	}
	if count != 0 {
		t.Fatalf("rollback row count = %d, want 0", count)
	}
}

func assertSeedRowCount(t *testing.T, ctx context.Context, database *sql.DB, query string, arguments ...any) {
	t.Helper()
	var count int
	if err := database.QueryRowContext(ctx, query, arguments...).Scan(&count); err != nil {
		t.Fatalf("count seed row (%s): %v", fmt.Sprint(arguments...), err)
	}
	if count != 1 {
		t.Fatalf("seed row count (%s) = %d, want 1", fmt.Sprint(arguments...), count)
	}
}
